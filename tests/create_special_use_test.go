package tests

import (
	"context"
	"testing"
	"time"

	"github.com/ProtonMail/gluon/connector"
	"github.com/ProtonMail/gluon/imap"
)

func TestCreateSpecialUse(t *testing.T) {
	runOneToOneTestWithAuth(t, defaultServerOptions(t), func(c *testConnection, _ *testSession) {
		c.C(`A001 CREATE MySent (USE (\Sent))`).OK(`A001`)
		c.C(`A002 CREATE "Old Mail" (use (\archive \all))`).OK(`A002`)

		c.C(`A003 LIST "" "*"`)
		c.S(
			`* LIST (\HasNoChildren \Unmarked) "/" "INBOX"`,
			`* LIST (\HasNoChildren \Sent \Unmarked) "/" "MySent"`,
			`* LIST (\All \Archive \HasNoChildren \Unmarked) "/" "Old Mail"`,
		)
		c.OK(`A003`)

		c.C(`A004 LIST (SPECIAL-USE) "" "*"`)
		c.S(
			`* LIST (\HasNoChildren \Sent \Unmarked) "/" "MySent"`,
			`* LIST (\All \Archive \HasNoChildren \Unmarked) "/" "Old Mail"`,
		)
		c.OK(`A004`)

		// The mailbox works like any other.
		c.doAppend(`MySent`, buildRFC5322TestLiteral(`To: 1@pm.me`)).expect("OK")
		c.C(`A005 SELECT MySent`).OK(`A005`)
		c.C(`A006 FETCH 1 (UID)`).S(`* 1 FETCH (UID 1)`).OK(`A006`)
	})
}

func TestCreateSpecialUseOnlyMarksTheNamedMailbox(t *testing.T) {
	runOneToOneTestWithAuth(t, defaultServerOptions(t), func(c *testConnection, _ *testSession) {
		c.C(`A001 CREATE Folders/Spam (USE (\Junk))`).OK(`A001`)

		c.C(`A002 LIST "" "Folders*"`)
		c.S(
			`* LIST (\HasChildren \Unmarked) "/" "Folders"`,
			`* LIST (\HasNoChildren \Junk \Unmarked) "/" "Folders/Spam"`,
		)
		c.OK(`A002`)
	})
}

func TestCreateSpecialUseWithoutAttributesIsAPlainCreate(t *testing.T) {
	runOneToOneTestWithAuth(t, defaultServerOptions(t), func(c *testConnection, _ *testSession) {
		c.C(`A001 CREATE Plain (USE ())`).OK(`A001`)
		c.C(`A002 LIST "" "Plain"`).S(`* LIST (\HasNoChildren \Unmarked) "/" "Plain"`).OK(`A002`)
	})
}

func TestCreateSpecialUseFailures(t *testing.T) {
	runOneToOneTestWithAuth(t, defaultServerOptions(t), func(c *testConnection, _ *testSession) {
		c.C(`A001 CREATE Starred (USE (\Important))`)
		c.Sx(`^A001 NO \[USEATTR\] .*unknown special-use attribute`)

		c.C(`A002 CREATE Both (USE (\Sent \Important))`)
		c.Sx(`^A002 NO \[USEATTR\]`)

		c.C(`A003 CREATE MySent (USE (\Sent))`).OK(`A003`)

		// The dummy connector allows one mailbox per special use.
		c.C(`A004 CREATE OtherSent (USE (\Sent))`)
		c.Sx(`^A004 NO \[USEATTR\] .*special use not supported`)

		c.C(`A005 CREATE MySent (USE (\Drafts))`).NO(`A005`)
		c.C(`A006 CREATE INBOX (USE (\Drafts))`).NO(`A006`)

		c.C(`A007 CREATE Box (USE (Sent))`).BAD(`A007`)
		c.C(`A008 CREATE Box (BOGUS (\Sent))`).BAD(`A008`)
		c.C(`A009 CREATE Box (USE (\Sent)) extra`).BAD(`A009`)

		// Nothing was created by the commands that failed.
		c.C(`A010 LIST "" "*"`)
		c.S(
			`* LIST (\HasNoChildren \Unmarked) "/" "INBOX"`,
			`* LIST (\HasNoChildren \Sent \Unmarked) "/" "MySent"`,
		)
		c.OK(`A010`)

		c.C(`A011 CAPABILITY`)
		c.Sx(`\* CAPABILITY .* CREATE-SPECIAL-USE `)
		c.OK(`A011`)
	})
}

func TestCreateSpecialUseNeedsLogin(t *testing.T) {
	runOneToOneTest(t, defaultServerOptions(t), func(c *testConnection, _ *testSession) {
		c.C(`A001 CREATE MySent (USE (\Sent))`).NO(`A001`)
	})
}

// plainConnector cannot give a mailbox a special use.
type plainConnector struct {
	*connector.Dummy
}

func (c *plainConnector) CreateSpecialUseMailbox(context.Context, connector.IMAPStateWrite, []string, imap.FlagSet) (imap.Mailbox, error) {
	return imap.Mailbox{}, connector.ErrSpecialUseNotSupported
}

type plainConnectorBuilder struct{}

func (plainConnectorBuilder) New(usernames []string, password []byte, period time.Duration, flags, permFlags, attrs imap.FlagSet) Connector {
	return &plainConnector{Dummy: connector.NewDummy(usernames, password, period, flags, permFlags, attrs)}
}

func TestCreateSpecialUseRefusedByTheConnector(t *testing.T) {
	runOneToOneTestWithAuth(t, defaultServerOptions(t, withConnectorBuilder(plainConnectorBuilder{})), func(c *testConnection, _ *testSession) {
		c.C(`A001 CREATE MySent (USE (\Sent))`)
		c.Sx(`^A001 NO \[USEATTR\] .*special use not supported`)

		c.C(`A002 CREATE Parent/MySent (USE (\Sent))`)
		c.Sx(`^A002 NO \[USEATTR\]`)

		// A plain CREATE still goes through the connector's CreateMailbox.
		c.C(`A003 CREATE MySent`).OK(`A003`)

		c.C(`A004 LIST "" "*"`)
		c.S(
			`* LIST (\HasNoChildren \Unmarked) "/" "INBOX"`,
			`* LIST (\HasNoChildren \Unmarked) "/" "MySent"`,
		)
		c.OK(`A004`)
	})
}

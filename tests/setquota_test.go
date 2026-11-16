package tests

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/ProtonMail/gluon/connector"
	"github.com/ProtonMail/gluon/imap"
	"github.com/ProtonMail/gluon/imap/command"
	"github.com/stretchr/testify/require"
)

func setTestQuota(s *testSession) {
	s.setQuota("user", "", imap.QuotaResource{ResourceName: "STORAGE", Usage: 100, Limit: 512}, imap.QuotaResource{ResourceName: "MESSAGE", Usage: 5, Limit: 1000})
}

func TestSetQuota(t *testing.T) {
	runOneToOneTestWithAuth(t, defaultServerOptions(t), func(c *testConnection, s *testSession) {
		setTestQuota(s)

		// The answer is the quota root as it is after the change.
		c.C(`A001 SETQUOTA "" (STORAGE 1024)`)
		c.S(`* QUOTA "" (STORAGE 100 1024 MESSAGE 5 1000)`)
		c.S(`A001 OK SETQUOTA`)

		c.C(`A002 setquota "" (message 50 storage 2048)`)
		c.S(`* QUOTA "" (STORAGE 100 2048 MESSAGE 5 50)`)
		c.S(`A002 OK SETQUOTA`)

		c.C(`A003 GETQUOTA ""`)
		c.S(`* QUOTA "" (STORAGE 100 2048 MESSAGE 5 50)`)
		c.S(`A003 OK GETQUOTA`)

		// An empty list changes nothing.
		c.C(`A004 SETQUOTA "" ()`)
		c.S(`* QUOTA "" (STORAGE 100 2048 MESSAGE 5 50)`)
		c.S(`A004 OK SETQUOTA`)

		c.C(`A005 SETQUOTA "" (STORAGE 0)`)
		c.S(`* QUOTA "" (STORAGE 100 0 MESSAGE 5 50)`)
		c.S(`A005 OK SETQUOTA`)
	})
}

func TestSetQuotaOfANamedRoot(t *testing.T) {
	runOneToOneTestWithAuth(t, defaultServerOptions(t), func(c *testConnection, s *testSession) {
		s.setQuota("user", "user-root", imap.QuotaResource{ResourceName: "STORAGE", Usage: 1, Limit: 10})

		c.C(`A001 SETQUOTA user-root (STORAGE 20)`)
		c.S(`* QUOTA "user-root" (STORAGE 1 20)`)
		c.S(`A001 OK SETQUOTA`)

		c.C(`A002 SETQUOTA {9+}` + "\r\n" + `user-root (STORAGE 30)`)
		c.S(`* QUOTA "user-root" (STORAGE 1 30)`)
		c.S(`A002 OK SETQUOTA`)
	})
}

func TestSetQuotaRefusedByTheDummyConnector(t *testing.T) {
	runOneToOneTestWithAuth(t, defaultServerOptions(t), func(c *testConnection, s *testSession) {
		setTestQuota(s)

		c.C(`A001 SETQUOTA missing-root (STORAGE 1024)`)
		c.Sx(`^A001 NO .*no such quota root`)

		c.C(`A002 SETQUOTA "" (MAILBOX 10)`)
		c.Sx(`^A002 NO .*no such quota resource MAILBOX`)

		// A limit that is refused leaves the limits before it in the same command unchanged too.
		c.C(`A003 SETQUOTA "" (STORAGE 4096 MAILBOX 10)`)
		c.Sx(`^A003 NO .*no such quota resource MAILBOX`)

		c.C(`A004 SETQUOTA "" (STORAGE 5000000000)`)
		c.Sx(`^A004 NO .*quota limit for STORAGE is too large`)

		c.C(`A005 GETQUOTA ""`)
		c.S(`* QUOTA "" (STORAGE 100 512 MESSAGE 5 1000)`)
		c.S(`A005 OK GETQUOTA`)
	})
}

// readOnlyQuotaConnector does not let a client change a quota.
type readOnlyQuotaConnector struct {
	*connector.Dummy
}

func (c *readOnlyQuotaConnector) SetQuotaLimits(context.Context, string, []imap.QuotaLimit) (*imap.QuotaRoot, error) {
	return nil, connector.ErrOperationNotAllowed
}

type readOnlyQuotaConnectorBuilder struct{}

func (readOnlyQuotaConnectorBuilder) New(usernames []string, password []byte, period time.Duration, flags, permFlags, attrs imap.FlagSet) Connector {
	return &readOnlyQuotaConnector{Dummy: connector.NewDummy(usernames, password, period, flags, permFlags, attrs)}
}

func TestSetQuotaRefusedByTheConnector(t *testing.T) {
	runOneToOneTestWithAuth(t, defaultServerOptions(t, withConnectorBuilder(readOnlyQuotaConnectorBuilder{})), func(c *testConnection, s *testSession) {
		setTestQuota(s)

		c.C(`A001 SETQUOTA "" (STORAGE 1024)`)
		c.Sx(`^A001 NO .*operation not allowed`)

		c.C(`A002 GETQUOTA ""`)
		c.S(`* QUOTA "" (STORAGE 100 512 MESSAGE 5 1000)`)
		c.S(`A002 OK GETQUOTA`)
	})
}

func TestSetQuotaFailures(t *testing.T) {
	runOneToOneTest(t, defaultServerOptions(t), func(c *testConnection, s *testSession) {
		c.C(`A001 SETQUOTA "" (STORAGE 1024)`).NO(`A001`)

		c.C(`A002 LOGIN user pass`).OK(`A002`)
		setTestQuota(s)

		for i, line := range []string{
			`SETQUOTA`,
			`SETQUOTA ""`,
			`SETQUOTA "" STORAGE 10`,
			`SETQUOTA "" (STORAGE)`,
			`SETQUOTA "" (STORAGE ten)`,
			`SETQUOTA "" (STORAGE -1)`,
			`SETQUOTA "" (STORAGE 10`,
			`SETQUOTA "" (STORAGE 10,MESSAGE 5)`,
			`SETQUOTA "" (STORAGE 10) extra`,
			`SETQUOTA "" (STORAGE 99999999999999999999999)`,
			`SETQUOTA (STORAGE 10)`,
		} {
			tag := fmt.Sprintf("B%03d", i)

			c.Cf(`%v %v`, tag, line)
			c.Sx(`^` + tag + ` BAD`)
		}

		c.C(`A003 GETQUOTA ""`)
		c.S(`* QUOTA "" (STORAGE 100 512 MESSAGE 5 1000)`)
		c.S(`A003 OK GETQUOTA`)

		c.C(`A004 CAPABILITY`)
		c.Sx(`\* CAPABILITY .* QUOTASET `)
		c.OK(`A004`)
	})
}

func TestParserSetQuota(t *testing.T) {
	cmd, err := parseCommand(`tag SETQUOTA "#user/alice" (storage 510 MESSAGE 9000000000)`)
	require.NoError(t, err)
	require.Equal(t, command.Command{Tag: "tag", Payload: &command.SetQuota{
		Root:   "#user/alice",
		Limits: []imap.QuotaLimit{{ResourceName: "STORAGE", Limit: 510}, {ResourceName: "MESSAGE", Limit: 9000000000}},
	}}, cmd)

	cmd, err = parseCommand(`tag SETQUOTA "" ()`)
	require.NoError(t, err)
	require.Equal(t, command.Command{Tag: "tag", Payload: &command.SetQuota{}}, cmd)
}

package tests

import (
	"fmt"
	"testing"

	"github.com/ProtonMail/gluon/imap/command"
	"github.com/stretchr/testify/require"
)

// The dummy connector's user owns every mailbox and has every right. Gluon adds the rights "c" and "d" of older
// clients to a response, as RFC 4314 asks.
const ownerRights = `lrswipkxteacd`

func TestACLMyRights(t *testing.T) {
	runOneToOneTestWithAuth(t, defaultServerOptions(t), func(c *testConnection, _ *testSession) {
		c.C(`A001 CREATE Work`).OK(`A001`)

		c.C(`A002 MYRIGHTS Work`).S(`* MYRIGHTS "Work" ` + ownerRights).OK(`A002`)
		c.C(`A003 myrights inbox`).S(`* MYRIGHTS "INBOX" ` + ownerRights).OK(`A003`)
	})
}

func TestACLGetAndSet(t *testing.T) {
	runOneToOneTestWithAuth(t, defaultServerOptions(t), func(c *testConnection, _ *testSession) {
		c.C(`A001 CREATE Work`).OK(`A001`)

		c.C(`A002 GETACL Work`).S(`* ACL "Work" user ` + ownerRights).OK(`A002`)

		c.C(`A003 SETACL Work fred lrs`).OK(`A003`)
		c.C(`A004 SETACL Work "some group" lr`).OK(`A004`)
		c.C(`A005 GETACL Work`).S(`* ACL "Work" user ` + ownerRights + ` fred lrs "some group" lr`).OK(`A005`)

		// Rights are added to and removed from the ones the identifier has.
		c.C(`A006 SETACL Work fred +wi`).OK(`A006`)
		c.C(`A007 GETACL Work`).S(`* ACL "Work" user ` + ownerRights + ` fred lrswi "some group" lr`).OK(`A007`)

		c.C(`A008 SETACL Work fred -rs`).OK(`A008`)
		c.C(`A009 SETACL Work fred +l`).OK(`A009`)
		c.C(`A010 GETACL Work`).S(`* ACL "Work" user ` + ownerRights + ` fred lwi "some group" lr`).OK(`A010`)

		// Without a sign the rights replace the ones the identifier has.
		c.C(`A011 SETACL Work fred r`).OK(`A011`)
		c.C(`A012 GETACL Work`).S(`* ACL "Work" user ` + ownerRights + ` fred r "some group" lr`).OK(`A012`)

		// Another mailbox has its own list.
		c.C(`A013 GETACL INBOX`).S(`* ACL "INBOX" user ` + ownerRights).OK(`A013`)
	})
}

func TestACLDelete(t *testing.T) {
	runOneToOneTestWithAuth(t, defaultServerOptions(t), func(c *testConnection, _ *testSession) {
		c.C(`A001 CREATE Work`).OK(`A001`)
		c.C(`A002 SETACL Work fred lrs`).OK(`A002`)
		c.C(`A003 SETACL Work anna lr`).OK(`A003`)
		c.C(`A004 SETACL Work tom l`).OK(`A004`)

		c.C(`A005 DELETEACL Work fred`).OK(`A005`)
		c.C(`A006 GETACL Work`).S(`* ACL "Work" user ` + ownerRights + ` anna lr tom l`).OK(`A006`)

		// An identifier that is left with no rights is removed too.
		c.C(`A007 SETACL Work anna ""`).OK(`A007`)
		c.C(`A008 SETACL Work tom -l`).OK(`A008`)
		c.C(`A009 GETACL Work`).S(`* ACL "Work" user ` + ownerRights).OK(`A009`)

		// Removing what is not there is not an error.
		c.C(`A010 DELETEACL Work nobody`).OK(`A010`)
		c.C(`A011 SETACL Work nobody -lrs`).OK(`A011`)
	})
}

func TestACLObsoleteRights(t *testing.T) {
	runOneToOneTestWithAuth(t, defaultServerOptions(t), func(c *testConnection, _ *testSession) {
		c.C(`A001 CREATE Work`).OK(`A001`)

		// "d" stands for "e" and "t", and is reported whenever one of them is set.
		c.C(`A002 SETACL Work fred lrd`).OK(`A002`)
		c.C(`A003 GETACL Work`).S(`* ACL "Work" user ` + ownerRights + ` fred lretd`).OK(`A003`)

		// "c" stands for "k" and "x".
		c.C(`A004 SETACL Work fred +c`).OK(`A004`)
		c.C(`A005 GETACL Work`).S(`* ACL "Work" user ` + ownerRights + ` fred lretkxcd`).OK(`A005`)

		c.C(`A006 SETACL Work fred -d`).OK(`A006`)
		c.C(`A007 GETACL Work`).S(`* ACL "Work" user ` + ownerRights + ` fred lrkxc`).OK(`A007`)

		c.C(`A008 SETACL Work fred -x`).OK(`A008`)
		c.C(`A009 GETACL Work`).S(`* ACL "Work" user ` + ownerRights + ` fred lrkc`).OK(`A009`)
	})
}

func TestACLListRights(t *testing.T) {
	runOneToOneTestWithAuth(t, defaultServerOptions(t), func(c *testConnection, _ *testSession) {
		c.C(`A001 CREATE Work`).OK(`A001`)

		// Another identifier has no right of its own and can be granted each right separately.
		c.C(`A002 LISTRIGHTS Work fred`).S(`* LISTRIGHTS "Work" fred "" l r s w i p k x t e a`).OK(`A002`)

		// The owner always has every right.
		c.C(`A003 LISTRIGHTS Work user`).S(`* LISTRIGHTS "Work" user lrswipkxtea`).OK(`A003`)

		c.C(`A004 LISTRIGHTS Work "some group"`).S(`* LISTRIGHTS "Work" "some group" "" l r s w i p k x t e a`).OK(`A004`)
	})
}

func TestACLRefusedByTheConnector(t *testing.T) {
	runOneToOneTestWithAuth(t, defaultServerOptions(t), func(c *testConnection, _ *testSession) {
		c.C(`A001 CREATE Work`).OK(`A001`)

		// The rights of the owner cannot be changed, and "z" is not a right.
		c.C(`A002 SETACL Work user lr`)
		c.Sx(`^A002 NO .*operation not allowed`)

		c.C(`A003 DELETEACL Work user`)
		c.Sx(`^A003 NO .*operation not allowed`)

		c.C(`A004 SETACL Work fred lrz`)
		c.Sx(`^A004 NO .*operation not allowed`)

		c.C(`A005 GETACL Work`).S(`* ACL "Work" user ` + ownerRights).OK(`A005`)
	})
}

func TestACLFailures(t *testing.T) {
	runOneToOneTest(t, defaultServerOptions(t), func(c *testConnection, _ *testSession) {
		c.C(`A001 MYRIGHTS INBOX`).NO(`A001`)
		c.C(`A002 GETACL INBOX`).NO(`A002`)

		c.C(`A003 LOGIN user pass`).OK(`A003`)

		for i, line := range []string{
			`SETACL Missing fred lrs`,
			`DELETEACL Missing fred`,
			`GETACL Missing`,
			`LISTRIGHTS Missing fred`,
			`MYRIGHTS Missing`,
			// The recovery mailbox only exists in Gluon.
			`MYRIGHTS "Recovered Messages"`,
			`GETACL "Recovered Messages"`,
		} {
			tag := fmt.Sprintf("B%03d", i)

			c.Cf(`%v %v`, tag, line)
			c.Sx(`^` + tag + ` NO`)
		}

		for i, line := range []string{
			`SETACL`,
			`SETACL INBOX`,
			`SETACL INBOX fred`,
			`SETACL INBOX fred LRS`,
			`SETACL INBOX fred l-r`,
			`SETACL INBOX fred "l r"`,
			`SETACL INBOX fred lrs extra`,
			`DELETEACL INBOX`,
			`DELETEACL INBOX fred extra`,
			`GETACL`,
			`GETACL INBOX extra`,
			`LISTRIGHTS INBOX`,
			`MYRIGHTS`,
			`MYRIGHTS INBOX extra`,
		} {
			tag := fmt.Sprintf("C%03d", i)

			c.Cf(`%v %v`, tag, line)
			c.Sx(`^` + tag + ` BAD`)
		}

		c.C(`A004 CAPABILITY`)
		c.Sx(`\* CAPABILITY ACL .* RIGHTS=texk `)
		c.OK(`A004`)
	})
}

func TestParserACL(t *testing.T) {
	for line, expected := range map[string]command.Payload{
		`tag SETACL INBOX/Drafts fred lrswi`: &command.SetACL{Mailbox: "INBOX/Drafts", Identifier: "fred", Rights: "lrswi"},
		`tag setacl inbox "a group" +lr`:     &command.SetACL{Mailbox: "INBOX", Identifier: "a group", Rights: "lr", Mode: command.ACLModeAdd},
		`tag SETACL Work -fred -l0`:          &command.SetACL{Mailbox: "Work", Identifier: "-fred", Rights: "l0", Mode: command.ACLModeRemove},
		`tag SETACL Work fred ""`:            &command.SetACL{Mailbox: "Work", Identifier: "fred"},
		`tag DELETEACL Work fred`:            &command.DeleteACL{Mailbox: "Work", Identifier: "fred"},
		`tag GETACL "Sent Items"`:            &command.GetACL{Mailbox: "Sent Items"},
		`tag LISTRIGHTS Work anyone`:         &command.ListRights{Mailbox: "Work", Identifier: "anyone"},
		`tag MYRIGHTS Work`:                  &command.MyRights{Mailbox: "Work"},
	} {
		cmd, err := parseCommand(line)
		require.NoError(t, err, line)
		require.Equal(t, command.Command{Tag: "tag", Payload: expected}, cmd, line)
	}
}

package tests

import (
	"fmt"
	"testing"
	"time"

	"github.com/ProtonMail/gluon/imap"
	"github.com/ProtonMail/gluon/imap/command"
	"github.com/stretchr/testify/require"
)

// createObjectIDMailbox creates a mailbox and two messages whose IDs the test chooses, as a connector does.
func createObjectIDMailbox(c *testConnection, s *testSession) {
	mboxID := imap.MailboxID("mbox-Archive_2026")

	require.NoError(s.tb, s.conns[s.userIDs["user"]].MailboxCreated(imap.Mailbox{
		ID:             mboxID,
		Name:           []string{"Archive"},
		Flags:          defaultFlags,
		PermanentFlags: defaultPermanentFlags,
		Attributes:     defaultAttributes,
	}))

	s.messageCreatedWithID("user", "msg-first_1", mboxID, []byte(buildRFC5322TestLiteral("To: 1@pm.me")), time.Now())
	s.messageCreatedWithID("user", "msg-second_2", mboxID, []byte(buildRFC5322TestLiteral("To: 2@pm.me")), time.Now())
	s.flush("user")
}

func TestObjectIDOfAMailbox(t *testing.T) {
	runOneToOneTestWithAuth(t, defaultServerOptions(t), func(c *testConnection, s *testSession) {
		createObjectIDMailbox(c, s)

		// The mailbox ID is the ID the connector gave the mailbox.
		c.C(`A001 STATUS Archive (MAILBOXID)`).S(`* STATUS "Archive" (MAILBOXID (mbox-Archive_2026))`).OK(`A001`)
		c.C(`A002 STATUS Archive (MESSAGES MAILBOXID UIDNEXT)`).S(`* STATUS "Archive" (MESSAGES 2 MAILBOXID (mbox-Archive_2026) UIDNEXT 3)`).OK(`A002`)

		c.C(`A003 SELECT Archive`)
		c.Se(`* OK [MAILBOXID (mbox-Archive_2026)] Mailbox ID`)
		c.OK(`A003`)

		// The selected mailbox reports the same ID.
		c.C(`A004 STATUS Archive (MAILBOXID)`).S(`* STATUS "Archive" (MAILBOXID (mbox-Archive_2026))`).OK(`A004`)

		c.C(`A005 EXAMINE Archive`)
		c.Se(`* OK [MAILBOXID (mbox-Archive_2026)]`)
		c.OK(`A005`)

		c.C(`A006 LIST "" "Archive" RETURN (STATUS (MAILBOXID))`)
		c.S(`* LIST (\HasNoChildren \Unmarked) "/" "Archive"`, `* STATUS "Archive" (MAILBOXID (mbox-Archive_2026))`)
		c.OK(`A006`)
	})
}

func TestObjectIDOfAMailboxSurvivesARename(t *testing.T) {
	runOneToOneTestWithAuth(t, defaultServerOptions(t), func(c *testConnection, s *testSession) {
		createObjectIDMailbox(c, s)

		c.C(`A001 RENAME Archive Old`).OK(`A001`)
		c.C(`A002 STATUS Old (MAILBOXID)`).S(`* STATUS "Old" (MAILBOXID (mbox-Archive_2026))`).OK(`A002`)
	})
}

func TestObjectIDOfACreatedMailbox(t *testing.T) {
	runOneToOneTestWithAuth(t, defaultServerOptions(t), func(c *testConnection, _ *testSession) {
		// CREATE reports the ID of the new mailbox, and of the named one when it creates the ones above it too.
		c.C(`A001 CREATE Work`)
		c.Sx(`^A001 OK \[MAILBOXID \([\w-]+\)\] CREATE`)

		c.C(`A002 CREATE Projects/Alpha/`)
		c.Sx(`^A002 OK \[MAILBOXID \([\w-]+\)\] CREATE`)

		c.C(`A003 CREATE MySent (USE (\Sent))`)
		c.Sx(`^A003 OK \[MAILBOXID \([\w-]+\)\] CREATE`)

		// A CREATE that fails reports none.
		c.C(`A004 CREATE Work`)
		c.Sx(`^A004 NO [^\[]`)

		// INBOX has the ID the dummy connector gives it.
		c.C(`A005 STATUS INBOX (MAILBOXID)`).S(`* STATUS "INBOX" (MAILBOXID (0))`).OK(`A005`)
	})
}

func TestObjectIDOfMessages(t *testing.T) {
	runOneToOneTestWithAuth(t, defaultServerOptions(t), func(c *testConnection, s *testSession) {
		createObjectIDMailbox(c, s)
		c.C(`A000 SELECT Archive`).OK(`A000`)

		c.C(`A001 FETCH 1:2 (EMAILID)`)
		c.S(`* 1 FETCH (EMAILID (msg-first_1))`, `* 2 FETCH (EMAILID (msg-second_2))`)
		c.OK(`A001`)

		c.C(`A002 UID FETCH 2 (emailid threadid)`).S(`* 2 FETCH (EMAILID (msg-second_2) THREADID NIL UID 2)`).OK(`A002`)
		c.C(`A003 FETCH 1 THREADID`).S(`* 1 FETCH (THREADID NIL)`).OK(`A003`)

		// A copy is the same message, so it has the same ID in the other mailbox.
		c.C(`A004 COPY 1 INBOX`).OK(`A004`)
		c.C(`A005 SELECT INBOX`).OK(`A005`)
		c.C(`A006 FETCH 1 (EMAILID)`).S(`* 1 FETCH (EMAILID (msg-first_1))`).OK(`A006`)
	})
}

func TestObjectIDSearch(t *testing.T) {
	runOneToOneTestWithAuth(t, defaultServerOptions(t), func(c *testConnection, s *testSession) {
		createObjectIDMailbox(c, s)
		c.C(`A000 SELECT Archive`).OK(`A000`)

		c.C(`A001 SEARCH EMAILID msg-second_2`).S(`* SEARCH 2`).OK(`A001`)
		c.C(`A002 UID SEARCH NOT EMAILID msg-second_2`).S(`* SEARCH 1`).OK(`A002`)
		c.C(`A003 SEARCH OR EMAILID msg-first_1 EMAILID msg-second_2`).S(`* SEARCH 1 2`).OK(`A003`)
		c.C(`A004 SEARCH RETURN (COUNT) emailid MSG-FIRST_1`).S(`* ESEARCH (TAG "A004") COUNT 0`).OK(`A004`)
		c.C(`A005 SEARCH EMAILID no-such-message`).S(`* SEARCH`).OK(`A005`)

		// No message has a thread.
		c.C(`A006 SEARCH THREADID msg-first_1`).S(`* SEARCH`).OK(`A006`)
		c.C(`A007 SEARCH NOT THREADID anything`).S(`* SEARCH 1 2`).OK(`A007`)
	})
}

func TestObjectIDThatTheConnectorIDCannotBe(t *testing.T) {
	runOneToOneTestWithAuth(t, defaultServerOptions(t), func(c *testConnection, s *testSession) {
		// These connector IDs hold characters an object ID may not have, so Gluon's own IDs are sent.
		mboxID := imap.MailboxID("mbox/with=odd+chars")

		require.NoError(t, s.conns[s.userIDs["user"]].MailboxCreated(imap.Mailbox{
			ID:             mboxID,
			Name:           []string{"Odd"},
			Flags:          defaultFlags,
			PermanentFlags: defaultPermanentFlags,
			Attributes:     defaultAttributes,
		}))

		s.messageCreatedWithID("user", "msg id with spaces==", mboxID, []byte(buildRFC5322TestLiteral("To: 1@pm.me")), time.Now())
		s.flush("user")

		c.C(`A001 STATUS Odd (MAILBOXID)`)
		c.Sx(`^\* STATUS "Odd" \(MAILBOXID \(\d+\)\)`)
		c.OK(`A001`)

		c.C(`A002 SELECT Odd`).OK(`A002`)

		c.C(`A003 FETCH 1 (EMAILID)`)
		c.Sx(`^\* 1 FETCH \(EMAILID \([0-9a-f-]{36}\)\)`)
		c.OK(`A003`)
	})
}

func TestObjectIDFailures(t *testing.T) {
	runOneToOneTest(t, defaultServerOptions(t), func(c *testConnection, s *testSession) {
		c.C(`A001 STATUS INBOX (MAILBOXID)`).NO(`A001`)

		c.C(`A002 LOGIN user pass`).OK(`A002`)

		c.C(`A003 STATUS Missing (MAILBOXID)`).NO(`A003`)
		c.C(`A004 FETCH 1 (EMAILID)`).NO(`A004`)

		createObjectIDMailbox(c, s)
		c.C(`A005 SELECT Archive`).OK(`A005`)

		for i, line := range []string{
			`SEARCH EMAILID`,
			`SEARCH EMAILID "msg first"`,
			`SEARCH EMAILID msg.first`,
			`SEARCH EMAILID msg@first`,
			`SEARCH THREADID`,
			`SEARCH EMAILID ` + fmt.Sprintf("%0256d", 1),
			`FETCH 1 (EMAILIDS)`,
			`STATUS Archive (MAILBOXIDS)`,
		} {
			tag := fmt.Sprintf("B%03d", i)

			c.Cf(`%v %v`, tag, line)
			c.Sx(`^` + tag + ` BAD`)
		}

		c.C(`A006 SEARCH EMAILID ` + fmt.Sprintf("%0255d", 1)).S(`* SEARCH`).OK(`A006`)

		c.C(`A007 CAPABILITY`)
		c.Sx(`\* CAPABILITY .* OBJECTID `)
		c.OK(`A007`)
	})
}

func TestParserObjectID(t *testing.T) {
	cmd, err := parseCommand(`tag UID FETCH 1 (EMAILID THREADID)`)
	require.NoError(t, err)
	require.Equal(t, []command.FetchAttribute{&command.FetchAttributeEmailID{}, &command.FetchAttributeThreadID{}},
		cmd.Payload.(*command.UID).Command.(*command.Fetch).Attributes)

	cmd, err = parseCommand(`tag SEARCH EMAILID M6d99ac3275bb4e THREADID T64b478a75b7ea9`)
	require.NoError(t, err)
	require.Equal(t, []command.SearchKey{
		&command.SearchKeyEmailID{Value: "M6d99ac3275bb4e"},
		&command.SearchKeyThreadID{Value: "T64b478a75b7ea9"},
	}, cmd.Payload.(*command.Search).Keys)

	cmd, err = parseCommand(`tag STATUS INBOX (MAILBOXID)`)
	require.NoError(t, err)
	require.Equal(t, []command.StatusAttribute{command.StatusAttributeMailboxID}, cmd.Payload.(*command.Status).Attributes)
	require.Equal(t, "MAILBOXID", command.StatusAttributeMailboxID.String())

	for id, valid := range map[string]bool{
		"M6d99ac3275bb4e": true,
		"a_b-C9":          true,
		"0":               true,
		"":                false,
		"a b":             false,
		"a.b":             false,
		"a=":              false,
		"é":               false,
	} {
		require.Equal(t, valid, imap.IsObjectID(id), id)
	}
}

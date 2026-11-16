package tests

import (
	"strings"
	"testing"

	"github.com/ProtonMail/gluon/imap"
	"github.com/stretchr/testify/require"
)

// selectLines selects a mailbox and returns the lines of the answer.
func selectLines(c *testConnection, tag, mailbox string) []string {
	c.Cf(`%v SELECT %v`, tag, mailbox)

	var lines []string

	for {
		line := string(c.read())
		lines = append(lines, line)

		if strings.HasPrefix(line, tag+" ") {
			return lines
		}
	}
}

func TestUIDOnlyEnable(t *testing.T) {
	runOneToOneTestWithAuth(t, defaultServerOptions(t), func(c *testConnection, _ *testSession) {
		c.C(`A001 ENABLE UIDONLY`).S(`* ENABLED UIDONLY`).OK(`A001`)
		c.C(`A002 ENABLE uidonly UTF8=ACCEPT`).S(`* ENABLED UTF8=ACCEPT`).OK(`A002`)
		c.C(`A003 ENABLE UIDONLY`).S(`* ENABLED`).OK(`A003`)
	})
}

func TestUIDOnlyRejectsSequenceNumbers(t *testing.T) {
	runOneToOneTestWithData(t, defaultServerOptions(t), func(c *testConnection, _ *testSession, mbox string, _ imap.MailboxID) {
		c.C(`A001 ENABLE UIDONLY`).S(`* ENABLED UIDONLY`).OK(`A001`)

		for _, cmd := range []string{
			`FETCH 1 (FLAGS)`,
			`FETCH 1:* (UID)`,
			`STORE 1 +FLAGS (\Seen)`,
			`COPY 1 INBOX`,
			`MOVE 1 INBOX`,
			`SEARCH ALL`,
			`SEARCH RETURN (COUNT) ALL`,
			`SORT (DATE) UTF-8 ALL`,
			`THREAD REFERENCES UTF-8 ALL`,
			// A sequence set is a search key that names messages by their sequence numbers.
			`UID SEARCH 1:5`,
			`UID SEARCH NOT 1:5`,
			`UID SEARCH OR SEEN 1:5`,
			`UID SEARCH (SEEN 1:5)`,
			`UID SORT (DATE) UTF-8 1:5`,
			`UID THREAD REFERENCES UTF-8 1:5`,
		} {
			c.Cf(`B001 %v`, cmd)
			c.Sx(`^B001 BAD \[UIDREQUIRED\]`)
		}

		// Nothing was copied, moved or changed.
		c.C(`A002 STATUS INBOX (MESSAGES)`).S(`* STATUS "INBOX" (MESSAGES 0)`).OK(`A002`)
		c.C(`A003 UID SEARCH SEEN`).S(`* SEARCH`).OK(`A003`)

		// The UID forms work.
		c.C(`A004 UID SEARCH UID 1:5`).S(`* SEARCH 1 2 3 4 5`).OK(`A004`)
		c.C(`A005 UID SEARCH RETURN (COUNT) ALL`).S(`* ESEARCH (TAG "A005") UID COUNT 100`).OK(`A005`)
		c.C(`A006 UID SORT (ARRIVAL) UTF-8 UID 1:3`).S(`* SORT 1 2 3`).OK(`A006`)
		c.Cf(`A007 STATUS %v (MESSAGES)`, mbox).Sxe(`MESSAGES 100`).OK(`A007`)
	})
}

func TestUIDOnlyResponsesNameMessagesByUID(t *testing.T) {
	runOneToOneTestWithAuth(t, defaultServerOptions(t), func(c *testConnection, _ *testSession) {
		for range 5 {
			c.doAppend(`INBOX`, buildRFC5322TestLiteral("Subject: uidonly\r\n\r\nbody")).expect("OK")
		}

		c.C(`A001 ENABLE UIDONLY`).S(`* ENABLED UIDONLY`).OK(`A001`)
		c.C(`A002 SELECT INBOX`).OK(`A002`)

		// After UID 1 is expunged, the message with UID 2 has the sequence number 1: the responses carry the UID.
		c.C(`A003 UID STORE 1 +FLAGS.SILENT (\Deleted)`).OK(`A003`)
		c.C(`A004 UID EXPUNGE 1`).S(`* VANISHED 1`).OK(`A004`)

		c.C(`A005 UID FETCH 2:3 (UID)`).S(`* 2 UIDFETCH (UID 2)`, `* 3 UIDFETCH (UID 3)`).OK(`A005`)
		c.C(`A006 UID STORE 2 +FLAGS (\Flagged)`).Sx(`^\* 2 UIDFETCH \(FLAGS \(.*\\Flagged.*\) UID 2\)`).OK(`A006`)

		c.C(`A007 UID STORE 2:3 +FLAGS.SILENT (\Deleted)`).OK(`A007`)
		c.C(`A008 EXPUNGE`).S(`* VANISHED 2`, `* VANISHED 3`).OK(`A008`)

		c.C(`A009 CREATE Other`).OK(`A009`)
		c.C(`A010 UID MOVE 4 Other`).Sx(`^\* OK \[COPYUID \d+ 4 1\]`, `^\* VANISHED 4\r\n`).OK(`A010`)
		c.C(`A011 UID COPY 5 Other`).Sx(`^A011 OK \[COPYUID \d+ 5 2\]`)
	})
}

func TestUIDOnlyChangesFromOtherSessions(t *testing.T) {
	runManyToOneTestWithAuth(t, defaultServerOptions(t), []int{1, 2}, func(c map[int]*testConnection, _ *testSession) {
		for range 3 {
			c[1].doAppend(`INBOX`, buildRFC5322TestLiteral("Subject: uidonly\r\n\r\nbody")).expect("OK")
		}

		c[1].C(`A001 ENABLE UIDONLY`).S(`* ENABLED UIDONLY`).OK(`A001`)
		c[1].C(`A002 SELECT INBOX`).OK(`A002`)
		c[2].C(`B001 SELECT INBOX`).OK(`B001`)

		c[2].C(`B002 UID STORE 1 +FLAGS.SILENT (\Deleted)`).OK(`B002`)
		c[2].C(`B003 EXPUNGE`).S(`* 1 EXPUNGE`).OK(`B003`)
		c[2].C(`B004 UID STORE 3 +FLAGS.SILENT (\Flagged)`).OK(`B004`)

		// The other session still gets sequence numbers; this one gets UIDs.
		c[1].C(`A003 NOOP`)
		c[1].Sx(`^\* 1 UIDFETCH \(FLAGS \(.*\\Deleted.*\)\)`, `^\* VANISHED 1\r\n`, `^\* 3 UIDFETCH \(FLAGS \(.*\\Flagged.*\)\)`)
		c[1].OK(`A003`)
	})
}

func TestUIDOnlySelectHasNoSequenceNumbers(t *testing.T) {
	runOneToOneTestWithAuth(t, defaultServerOptions(t), func(c *testConnection, _ *testSession) {
		c.doAppend(`INBOX`, buildRFC5322TestLiteral("Subject: uidonly\r\n\r\nbody"), `\Seen`).expect("OK")
		c.doAppend(`INBOX`, buildRFC5322TestLiteral("Subject: uidonly\r\n\r\nbody")).expect("OK")

		require.Contains(t, strings.Join(selectLines(c, "A001", "INBOX"), ""), "[UNSEEN 2]")

		// RFC 9586: once UIDONLY is enabled, no response gives a message sequence number.
		c.C(`A002 ENABLE UIDONLY`).S(`* ENABLED UIDONLY`).OK(`A002`)
		require.NotContains(t, strings.Join(selectLines(c, "A003", "INBOX"), ""), "UNSEEN")
	})
}

func TestUIDOnlyEndsWithTheLogin(t *testing.T) {
	runOneToOneTestWithData(t, defaultServerOptions(t), func(c *testConnection, _ *testSession, mbox string, _ imap.MailboxID) {
		c.C(`A001 ENABLE UIDONLY`).S(`* ENABLED UIDONLY`).OK(`A001`)
		c.C(`A002 FETCH 1 (UID)`).Sx(`^A002 BAD \[UIDREQUIRED\]`)

		c.C(`A003 UNAUTHENTICATE`).OK(`A003`)
		c.C(`A004 LOGIN user pass`).OK(`A004`)
		c.Cf(`A005 SELECT %v`, mbox).OK(`A005`)

		c.C(`A006 FETCH 1 (UID)`).S(`* 1 FETCH (UID 1)`).OK(`A006`)
	})
}

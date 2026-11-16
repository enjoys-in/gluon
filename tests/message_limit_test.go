package tests

import (
	"fmt"
	"strings"
	"testing"

	"github.com/ProtonMail/gluon/imap"
	"github.com/ProtonMail/gluon/imap/command"
	"github.com/stretchr/testify/require"
)

type messageLimitOption struct {
	limit int
}

func (opt messageLimitOption) apply(options *serverOptions) {
	options.messageLimit = opt.limit
}

// withMessageLimit limits how many messages one command processes (RFC 9738).
func withMessageLimit(limit int) serverOption {
	return messageLimitOption{limit: limit}
}

// fetchedUIDs are the untagged responses to a FETCH of the UIDs from first to last, one message per sequence number.
func fetchedUIDs(uids ...int) []string {
	lines := make([]string, 0, len(uids))

	for _, uid := range uids {
		lines = append(lines, fmt.Sprintf(`* %d FETCH (UID %d)`, uid, uid))
	}

	return lines
}

func uidRange(first, last int) []int {
	var uids []int

	for uid := first; uid <= last; uid++ {
		uids = append(uids, uid)
	}

	return uids
}

func searchResult(uids ...int) string {
	res := `* SEARCH`

	for _, uid := range uids {
		res += fmt.Sprintf(" %d", uid)
	}

	return res
}

func TestMessageLimitCapability(t *testing.T) {
	runOneToOneTest(t, defaultServerOptions(t, withMessageLimit(10)), func(c *testConnection, _ *testSession) {
		// It is advertised to clients that are logged in.
		c.C(`A001 CAPABILITY`)
		require.NotContains(t, string(c.read()), "MESSAGELIMIT")
		c.OK(`A001`)

		c.C(`A002 LOGIN user pass`).OK(`A002`)

		c.C(`A003 CAPABILITY`)
		require.Contains(t, string(c.read()), " MESSAGELIMIT=10")
		c.OK(`A003`)
	})

	runOneToOneTestWithAuth(t, defaultServerOptions(t), func(c *testConnection, _ *testSession) {
		c.C(`A001 CAPABILITY`)
		require.NotContains(t, string(c.read()), "MESSAGELIMIT")
		c.OK(`A001`)
	})
}

func TestMessageLimitIsOffByDefault(t *testing.T) {
	runOneToOneTestWithData(t, defaultServerOptions(t), func(c *testConnection, _ *testSession, _ string, _ imap.MailboxID) {
		c.C(`A001 FETCH 1:100 (UID)`).S(fetchedUIDs(uidRange(1, 100)...)...)
		c.Sx(`^A001 OK [^\[]`)
	})
}

func TestMessageLimitFetch(t *testing.T) {
	runOneToOneTestWithData(t, defaultServerOptions(t, withMessageLimit(10)), func(c *testConnection, _ *testSession, _ string, _ imap.MailboxID) {
		// Only the messages with the highest UIDs are fetched, and the response says where the command stopped.
		c.C(`A001 FETCH 1:100 (UID)`).S(fetchedUIDs(uidRange(91, 100)...)...)
		c.OK(`A001`, `MESSAGELIMIT 10 91`)

		c.C(`A002 UID FETCH 1:* (UID)`).S(fetchedUIDs(uidRange(91, 100)...)...)
		c.OK(`A002`, `MESSAGELIMIT 10 91`)

		// The client carries on below the UID it was told.
		c.C(`A003 UID FETCH 81:90 (UID)`).S(fetchedUIDs(uidRange(81, 90)...)...)
		c.Sx(`^A003 OK [^\[]`)

		// A set that is not one range is cut at the same place.
		c.C(`A004 FETCH 1:3,5:7,50:57 (UID)`).S(fetchedUIDs(append([]int{6, 7}, uidRange(50, 57)...)...)...)
		c.OK(`A004`, `MESSAGELIMIT 10 6`)

		// A set of no more than the limit is fetched whole.
		c.C(`A005 FETCH 91:100 (UID)`).S(fetchedUIDs(uidRange(91, 100)...)...)
		c.Sx(`^A005 OK [^\[]`)
	})
}

func TestMessageLimitSearch(t *testing.T) {
	runOneToOneTestWithData(t, defaultServerOptions(t, withMessageLimit(10)), func(c *testConnection, _ *testSession, _ string, _ imap.MailboxID) {
		c.C(`A001 SEARCH ALL`).S(searchResult(uidRange(91, 100)...))
		c.OK(`A001`, `MESSAGELIMIT 10 91`)

		c.C(`A002 UID SEARCH UIDBEFORE 91`).S(searchResult(uidRange(81, 90)...))
		c.OK(`A002`, `MESSAGELIMIT 10 81`)

		c.C(`A003 UID SEARCH UIDAFTER 95`).S(searchResult(uidRange(96, 100)...))
		c.Sx(`^A003 OK [^\[]`)

		c.C(`A004 UID SEARCH UIDBEFORE 1`).S(`* SEARCH`)
		c.Sx(`^A004 OK [^\[]`)

		c.C(`A005 SEARCH RETURN (COUNT MAX) ALL`).S(`* ESEARCH (TAG "A005") COUNT 10 MAX 100`)
		c.OK(`A005`, `MESSAGELIMIT 10 91`)
	})
}

func TestMessageLimitSearchResult(t *testing.T) {
	runOneToOneTestWithData(t, defaultServerOptions(t, withMessageLimit(10)), func(c *testConnection, _ *testSession, _ string, _ imap.MailboxID) {
		// RFC 9738: what is saved in "$" is cut to the limit as well.
		c.C(`A001 SEARCH RETURN (SAVE) ALL`)
		c.OK(`A001`, `MESSAGELIMIT 10 91`)

		c.C(`A002 FETCH $ (UID)`).S(fetchedUIDs(uidRange(91, 100)...)...)
		c.Sx(`^A002 OK [^\[]`)
	})
}

func TestMessageLimitStore(t *testing.T) {
	runOneToOneTestWithData(t, defaultServerOptions(t, withMessageLimit(10)), func(c *testConnection, _ *testSession, _ string, _ imap.MailboxID) {
		c.C(`A001 STORE 1:100 +FLAGS (\Flagged)`).Sx(repeat(`^\* \d+ FETCH \(FLAGS \(.*\\Flagged`, 10)...)
		c.OK(`A001`, `MESSAGELIMIT 10 91`)

		// Only the newest messages were changed.
		c.C(`A002 UID SEARCH FLAGGED`).S(searchResult(uidRange(91, 100)...))
		c.Sx(`^A002 OK [^\[]`)

		c.C(`A003 UID STORE 91:100 -FLAGS.SILENT (\Flagged)`)
		c.Sx(`^A003 OK [^\[]`)
	})
}

func TestMessageLimitCopyAndMove(t *testing.T) {
	runOneToOneTestWithData(t, defaultServerOptions(t, withMessageLimit(10)), func(c *testConnection, _ *testSession, mbox string, _ imap.MailboxID) {
		c.C(`A001 CREATE dest`).OK(`A001`)

		// A COPY is all or nothing, so one that is over the limit copies nothing.
		c.C(`A002 COPY 1:100 dest`)
		c.Sx(`^A002 NO \[MESSAGELIMIT 10\]`)
		c.C(`A003 STATUS dest (MESSAGES)`).S(`* STATUS "dest" (MESSAGES 0)`).OK(`A003`)

		c.C(`A004 UID COPY 91:100 dest`)
		c.Sx(`^A004 OK \[COPYUID \d+ 91:100 1:10\]`)

		// A MOVE moves the newest messages and says where it stopped.
		c.C(`A005 UID MOVE 1:90 dest`)
		c.Sx(append(repeat(`^\* \d+ EXPUNGE`, 10), `^\* OK \[COPYUID \d+ 81:90 11:20\]`)...)
		c.OK(`A005`, `MESSAGELIMIT 10 81`)

		c.Cf(`A006 STATUS %v (MESSAGES)`, mbox).Sxe(`MESSAGES 90`).OK(`A006`)
		c.C(`A007 STATUS dest (MESSAGES)`).S(`* STATUS "dest" (MESSAGES 20)`).OK(`A007`)
	})
}

func TestMessageLimitUIDExpunge(t *testing.T) {
	runOneToOneTestWithData(t, defaultServerOptions(t, withMessageLimit(10)), func(c *testConnection, _ *testSession, mbox string, _ imap.MailboxID) {
		c.C(`A001 STORE 1:10 +FLAGS.SILENT (\Deleted)`).OK(`A001`)
		c.C(`A002 STORE 11:20 +FLAGS.SILENT (\Deleted)`).OK(`A002`)

		// Of the messages marked \Deleted only the newest are expunged.
		c.C(`A003 UID EXPUNGE 1:*`).Sx(repeat(`^\* \d+ EXPUNGE`, 10)...)
		c.OK(`A003`, `MESSAGELIMIT 10 11`)

		c.C(`A004 UID EXPUNGE 1:*`).Sx(repeat(`^\* \d+ EXPUNGE`, 10)...)
		c.Sx(`^A004 OK [^\[]`)

		c.Cf(`A005 STATUS %v (MESSAGES)`, mbox).Sxe(`MESSAGES 80`).OK(`A005`)
	})
}

func TestMessageLimitExpungeIsNotLimited(t *testing.T) {
	runOneToOneTestWithData(t, defaultServerOptions(t, withMessageLimit(10)), func(c *testConnection, _ *testSession, mbox string, _ imap.MailboxID) {
		for i, set := range []string{"1:10", "11:20", "21:30"} {
			c.Cf(`A00%d STORE %v +FLAGS.SILENT (\Deleted)`, i+1, set).OK(fmt.Sprintf("A00%d", i+1))
		}

		c.C(`A004 EXPUNGE`).Sx(repeat(`^\* \d+ EXPUNGE`, 30)...)
		c.Sx(`^A004 OK [^\[]`)

		c.Cf(`A005 STATUS %v (MESSAGES)`, mbox).Sxe(`MESSAGES 70`).OK(`A005`)
	})
}

func TestMessageLimitAppend(t *testing.T) {
	runOneToOneTestWithAuth(t, defaultServerOptions(t, withMessageLimit(3)), func(c *testConnection, _ *testSession) {
		literal := buildRFC5322TestLiteral("Subject: limit\r\n\r\nbody")
		message := fmt.Sprintf("{%v+}\r\n%v", len(literal), literal)

		// The messages of one APPEND are all appended or none is.
		c.Cf("A001 APPEND INBOX %v", strings.Join([]string{message, message, message, message}, " "))
		c.Sx(`^A001 NO \[MESSAGELIMIT 3\]`)
		c.C(`A002 STATUS INBOX (MESSAGES)`).S(`* STATUS "INBOX" (MESSAGES 0)`).OK(`A002`)

		c.Cf("A003 APPEND INBOX %v", strings.Join([]string{message, message, message}, " "))
		c.Sx(`^A003 OK \[APPENDUID \d+ 1:3\]`)
		c.C(`A004 STATUS INBOX (MESSAGES)`).S(`* STATUS "INBOX" (MESSAGES 3)`).OK(`A004`)
	})
}

func TestMessageLimitSearchKeyFailures(t *testing.T) {
	runOneToOneTestWithData(t, defaultServerOptions(t), func(c *testConnection, _ *testSession, _ string, _ imap.MailboxID) {
		c.C(`A001 UID SEARCH UIDAFTER 0`).BAD(`A001`)
		c.C(`A002 UID SEARCH UIDBEFORE`).BAD(`A002`)
		c.C(`A003 UID SEARCH UIDAFTER abc`).BAD(`A003`)

		// They work without the limit.
		c.C(`A004 UID SEARCH UIDAFTER 98`).S(`* SEARCH 99 100`).OK(`A004`)
		c.C(`A005 UID SEARCH UIDBEFORE 3`).S(`* SEARCH 1 2`).OK(`A005`)
	})
}

func TestParserSearchUIDBounds(t *testing.T) {
	cmd, err := parseCommand(`tag SEARCH UIDAFTER 5 uidbefore 9`)
	require.NoError(t, err)
	require.Equal(t, &command.Search{Keys: []command.SearchKey{
		&command.SearchKeyUID{SeqSet: []command.SeqRange{{Begin: 6, End: command.SeqNumValueAsterisk}}},
		&command.SearchKeyUID{SeqSet: []command.SeqRange{{Begin: 1, End: 8}}},
	}}, cmd.Payload)

	// There is no UID below 1.
	cmd, err = parseCommand(`tag SEARCH UIDBEFORE 1`)
	require.NoError(t, err)
	require.Equal(t, &command.Search{Keys: []command.SearchKey{&command.SearchKeyNot{Key: &command.SearchKeyAll{}}}}, cmd.Payload)

	for _, line := range []string{
		`tag SEARCH UIDAFTER`,
		`tag SEARCH UIDAFTER 0`,
		`tag SEARCH UIDBEFORE x`,
	} {
		_, err := parseCommand(line)
		require.Error(t, err, line)
	}
}

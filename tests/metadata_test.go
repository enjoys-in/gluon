package tests

import (
	"fmt"
	"strings"
	"testing"

	"github.com/ProtonMail/gluon/imap"
	"github.com/ProtonMail/gluon/imap/command"
	"github.com/stretchr/testify/require"
)

func TestMetadataSetAndGet(t *testing.T) {
	runOneToOneTestWithAuth(t, defaultServerOptions(t), func(c *testConnection, _ *testSession) {
		c.C(`A001 SETMETADATA INBOX (/private/comment "My own comment")`).S(`A001 OK SETMETADATA`)

		c.C(`A002 GETMETADATA INBOX /private/comment`)
		c.S(`* METADATA "INBOX" (/private/comment "My own comment")`)
		c.S(`A002 OK GETMETADATA`)

		c.C(`A003 SETMETADATA INBOX (/shared/comment "Shared comment" /private/vendor/example/color "blue")`).OK(`A003`)

		// The entries come back in the order they were asked for.
		c.C(`A004 GETMETADATA INBOX (/shared/comment /private/vendor/example/color /private/comment)`)
		c.S(`* METADATA "INBOX" (/shared/comment "Shared comment" /private/vendor/example/color "blue" /private/comment "My own comment")`)
		c.OK(`A004`)

		// Entry names are case-insensitive.
		c.C(`A005 SETMETADATA INBOX (/Private/Comment "Changed")`).OK(`A005`)
		c.C(`A006 GETMETADATA INBOX /PRIVATE/COMMENT`).S(`* METADATA "INBOX" (/private/comment "Changed")`).OK(`A006`)
	})
}

func TestMetadataEntryThatDoesNotExist(t *testing.T) {
	runOneToOneTestWithAuth(t, defaultServerOptions(t), func(c *testConnection, _ *testSession) {
		c.C(`A001 GETMETADATA INBOX /shared/comment`).S(`* METADATA "INBOX" (/shared/comment NIL)`).OK(`A001`)

		c.C(`A002 SETMETADATA INBOX (/private/comment "My own comment")`).OK(`A002`)
		c.C(`A003 GETMETADATA INBOX (/shared/comment /private/comment)`)
		c.S(`* METADATA "INBOX" (/shared/comment NIL /private/comment "My own comment")`)
		c.OK(`A003`)

		// NIL removes an entry, and removing one that is not there is not an error.
		c.C(`A004 SETMETADATA INBOX (/private/comment NIL /shared/comment NIL)`).OK(`A004`)
		c.C(`A005 GETMETADATA INBOX /private/comment`).S(`* METADATA "INBOX" (/private/comment NIL)`).OK(`A005`)
	})
}

func TestMetadataOfTheServer(t *testing.T) {
	runOneToOneTestWithAuth(t, defaultServerOptions(t), func(c *testConnection, _ *testSession) {
		// The empty mailbox name stands for the server.
		c.C(`A001 SETMETADATA "" (/shared/admin "mailto:admin@example.com")`).OK(`A001`)
		c.C(`A002 SETMETADATA INBOX (/shared/comment "About the inbox")`).OK(`A002`)

		c.C(`A003 GETMETADATA "" (/shared/admin /shared/comment)`)
		c.S(`* METADATA "" (/shared/admin "mailto:admin@example.com" /shared/comment NIL)`)
		c.OK(`A003`)

		c.C(`A004 GETMETADATA INBOX (/shared/admin /shared/comment)`)
		c.S(`* METADATA "INBOX" (/shared/admin NIL /shared/comment "About the inbox")`)
		c.OK(`A004`)
	})
}

func TestMetadataDepth(t *testing.T) {
	runOneToOneTestWithAuth(t, defaultServerOptions(t), func(c *testConnection, _ *testSession) {
		c.C(`A001 SETMETADATA INBOX (/private/filters/values/small "SMALLER 5000" /private/filters/values/boss "FROM \"boss@example.com\"")`).OK(`A001`)
		c.C(`A002 SETMETADATA INBOX (/private/filters/values/deep/er "deep" /private/filters "top")`).OK(`A002`)

		// Without DEPTH only the entry itself is returned.
		c.C(`A003 GETMETADATA INBOX /private/filters`).S(`* METADATA "INBOX" (/private/filters "top")`).OK(`A003`)
		c.C(`A004 GETMETADATA INBOX (DEPTH 0) /private/filters`).S(`* METADATA "INBOX" (/private/filters "top")`).OK(`A004`)

		// DEPTH 1 adds the entries right below it. The entry itself has no value here and is left out.
		c.C(`A005 GETMETADATA INBOX (DEPTH 1) (/private/filters/values)`)
		c.S(`* METADATA "INBOX" (/private/filters/values/boss "FROM \"boss@example.com\"" /private/filters/values/small "SMALLER 5000")`)
		c.OK(`A005`)

		// DEPTH infinity adds everything below it.
		c.C(`A006 GETMETADATA INBOX (DEPTH infinity) /private/filters`)
		c.S(`* METADATA "INBOX" (/private/filters "top" /private/filters/values/boss "FROM \"boss@example.com\"" /private/filters/values/deep/er "deep" /private/filters/values/small "SMALLER 5000")`)
		c.OK(`A006`)

		// A name that only starts the same is not below the entry.
		c.C(`A007 SETMETADATA INBOX (/private/filtersextra "no")`).OK(`A007`)
		c.C(`A008 GETMETADATA INBOX (DEPTH 1) /private/filters`).S(`* METADATA "INBOX" (/private/filters "top")`).OK(`A008`)

		// Nothing to report: no METADATA response.
		c.C(`A009 GETMETADATA INBOX (DEPTH infinity) /shared/nothing`).Sx(`^A009 OK`)
	})
}

func TestMetadataMaxSize(t *testing.T) {
	runOneToOneTestWithAuth(t, defaultServerOptions(t), func(c *testConnection, _ *testSession) {
		long := strings.Repeat("x", 600)

		c.Cf(`A001 SETMETADATA INBOX (/shared/comment "%v" /private/comment "My own comment" /private/note "%v")`, long, long[:300]).OK(`A001`)

		// Values over the size are left out and the OK says how big the biggest one was.
		c.C(`A002 GETMETADATA INBOX (MAXSIZE 100) (/shared/comment /private/comment /private/note)`)
		c.S(`* METADATA "INBOX" (/private/comment "My own comment")`)
		c.S(`A002 OK [METADATA LONGENTRIES 600] GETMETADATA`)

		// The grammar of RFC 5464 has the options before the mailbox, its examples after it.
		c.C(`A003 GETMETADATA (MAXSIZE 100) INBOX (/shared/comment /private/comment)`)
		c.S(`* METADATA "INBOX" (/private/comment "My own comment")`)
		c.S(`A003 OK [METADATA LONGENTRIES 600] GETMETADATA`)

		c.C(`A004 GETMETADATA INBOX (MAXSIZE 14 DEPTH infinity) /private/comment`)
		c.S(`* METADATA "INBOX" (/private/comment "My own comment")`)
		c.S(`A004 OK GETMETADATA`)

		c.C(`A005 GETMETADATA (DEPTH 1 MAXSIZE 13) INBOX /private/comment`)
		c.S(`A005 OK [METADATA LONGENTRIES 14] GETMETADATA`)
	})
}

func TestMetadataValues(t *testing.T) {
	runOneToOneTestWithAuth(t, defaultServerOptions(t), func(c *testConnection, _ *testSession) {
		c.C(`A001 SETMETADATA INBOX (/private/quoted "say \"hi\" \\ bye" /private/empty "")`).OK(`A001`)
		c.C("A002 SETMETADATA INBOX (/private/lines {12+}\r\nline1\r\nline2)").OK(`A002`)
		c.C("A003 SETMETADATA INBOX (/private/binary ~{4+}\r\na\x00b\x01)").OK(`A003`)

		c.C(`A004 GETMETADATA INBOX (/private/quoted /private/empty)`)
		c.S(`* METADATA "INBOX" (/private/quoted "say \"hi\" \\ bye" /private/empty "")`)
		c.OK(`A004`)

		// Text a quoted string cannot hold is sent as a literal, and a value with a zero byte as a binary one.
		c.C(`A005 GETMETADATA INBOX /private/lines`)
		c.S("* METADATA \"INBOX\" (/private/lines {12}\r\nline1\r\nline2)")
		c.OK(`A005`)

		c.C(`A006 GETMETADATA INBOX /private/binary`)
		c.S("* METADATA \"INBOX\" (/private/binary ~{4}\r\na\x00b\x01)")
		c.OK(`A006`)
	})
}

func TestMetadataFollowsTheMailbox(t *testing.T) {
	runOneToOneTestWithAuth(t, defaultServerOptions(t), func(c *testConnection, s *testSession) {
		c.C(`A001 CREATE Work`).OK(`A001`)
		c.C(`A002 SETMETADATA Work (/shared/comment "Work things")`).OK(`A002`)

		c.C(`A003 RENAME Work Old`).OK(`A003`)
		c.C(`A004 GETMETADATA Old /shared/comment`).S(`* METADATA "Old" (/shared/comment "Work things")`).OK(`A004`)

		// Let the connector report each change before the name is used again.
		s.flush("user")
		c.C(`A005 DELETE Old`).OK(`A005`)
		s.flush("user")
		c.C(`A006 CREATE Old`).OK(`A006`)
		c.C(`A007 GETMETADATA Old /shared/comment`).S(`* METADATA "Old" (/shared/comment NIL)`).OK(`A007`)
	})
}

func TestMetadataRefusedByTheConnector(t *testing.T) {
	runOneToOneTestWithAuth(t, defaultServerOptions(t), func(c *testConnection, s *testSession) {
		c.C(`A001 SETMETADATA INBOX (/private/comment "kept")`).OK(`A001`)

		// The dummy connector keeps values of up to 1024 bytes.
		c.Cf(`A002 SETMETADATA INBOX (/private/comment "changed" /shared/comment "%v")`, strings.Repeat("x", 1025))
		c.Sx(`^A002 NO \[METADATA MAXSIZE 1024\]`)

		c.Cf(`A003 SETMETADATA INBOX (/shared/comment "%v")`, strings.Repeat("x", 1024)).OK(`A003`)

		// It keeps ten annotations for a mailbox.
		c.C(`A004 SETMETADATA INBOX (/private/a/1 "1" /private/a/2 "2" /private/a/3 "3" /private/a/4 "4" /private/a/5 "5" /private/a/6 "6" /private/a/7 "7" /private/a/8 "8")`).OK(`A004`)
		c.C(`A005 SETMETADATA INBOX (/private/comment "changed" /private/a/9 "9")`)
		c.Sx(`^A005 NO \[METADATA TOOMANY\]`)

		// Replacing and removing still work when the list is full.
		c.C(`A006 SETMETADATA INBOX (/private/a/1 NIL /private/a/9 "9")`).OK(`A006`)

		// A command that is refused changes nothing.
		c.C(`A007 GETMETADATA INBOX (/private/comment /private/a/1 /private/a/9)`)
		c.S(`* METADATA "INBOX" (/private/comment "kept" /private/a/1 NIL /private/a/9 "9")`)
		c.OK(`A007`)

		s.conns[s.userIDs["user"]].SetPrivateMetadataDisabled(true)

		c.C(`A008 SETMETADATA INBOX (/shared/note "shared" /private/note "private")`)
		c.Sx(`^A008 NO \[METADATA NOPRIVATE\]`)

		c.C(`A009 SETMETADATA INBOX (/shared/comment "shared")`).OK(`A009`)
	})
}

func TestMetadataFailures(t *testing.T) {
	runOneToOneTest(t, defaultServerOptions(t), func(c *testConnection, _ *testSession) {
		c.C(`A001 GETMETADATA INBOX /private/comment`).NO(`A001`)
		c.C(`A002 SETMETADATA INBOX (/private/comment "x")`).NO(`A002`)

		c.C(`A003 LOGIN user pass`).OK(`A003`)

		c.C(`A004 GETMETADATA Missing /private/comment`).NO(`A004`)
		c.C(`A005 SETMETADATA Missing (/private/comment "x")`).NO(`A005`)
		c.C(`A006 GETMETADATA "Recovered Messages" /private/comment`).NO(`A006`)

		for i, line := range []string{
			// Entry names that RFC 5464 does not allow.
			`GETMETADATA INBOX /comment`,
			`GETMETADATA INBOX /private`,
			`GETMETADATA INBOX /private/`,
			`GETMETADATA INBOX /private//comment`,
			`GETMETADATA INBOX /private/comment/`,
			`GETMETADATA INBOX "/private/com*ment"`,
			`GETMETADATA INBOX "/private/com%ment"`,
			`GETMETADATA INBOX /public/comment`,
			`GETMETADATA INBOX private/comment`,
			`GETMETADATA INBOX /private/vendor/example`,
			`SETMETADATA INBOX (/comment "x")`,
			`SETMETADATA INBOX (/shared/vendor/example "x")`,
			// Commands that are not complete or not understood.
			`GETMETADATA`,
			`GETMETADATA INBOX`,
			`GETMETADATA INBOX ()`,
			`GETMETADATA INBOX (DEPTH 2) /private/comment`,
			`GETMETADATA INBOX (DEPTH) /private/comment`,
			`GETMETADATA INBOX (MAXSIZE big) /private/comment`,
			`GETMETADATA INBOX (BOGUS 1) /private/comment`,
			`GETMETADATA INBOX (DEPTH 1)`,
			`GETMETADATA INBOX /private/comment extra`,
			`SETMETADATA`,
			`SETMETADATA INBOX`,
			`SETMETADATA INBOX ()`,
			`SETMETADATA INBOX (/private/comment)`,
			`SETMETADATA INBOX /private/comment "x"`,
			`SETMETADATA INBOX (/private/comment x)`,
			`SETMETADATA INBOX (/private/comment "x") extra`,
		} {
			tag := fmt.Sprintf("B%03d", i)

			c.Cf(`%v %v`, tag, line)
			c.Sx(`^` + tag + ` BAD`)

			// The server closes a session after too many errors in a row.
			c.Cf(`%v NOOP`, tag).OK(tag)
		}

		c.C(`A007 CAPABILITY`)
		c.Sx(`\* CAPABILITY .* METADATA `)
		c.OK(`A007`)
	})
}

func TestParserMetadata(t *testing.T) {
	cmd, err := parseCommand(`tag SETMETADATA INBOX (/private/comment "My comment" /Shared/Comment NIL /private/empty "")`)
	require.NoError(t, err)
	require.Equal(t, command.Command{Tag: "tag", Payload: &command.SetMetadata{
		Mailbox: "INBOX",
		Entries: []imap.MetadataEntry{
			{Name: "/private/comment", Value: []byte("My comment")},
			{Name: "/shared/comment"},
			{Name: "/private/empty", Value: []byte{}},
		},
	}}, cmd)

	for line, expected := range map[string]*command.GetMetadata{
		`tag GETMETADATA "" /shared/comment`: {Entries: []string{"/shared/comment"}},
		`tag GETMETADATA INBOX (/shared/comment /private/comment)`: {
			Mailbox: "INBOX", Entries: []string{"/shared/comment", "/private/comment"},
		},
		`tag GETMETADATA INBOX (MAXSIZE 1024) (/shared/comment)`: {
			Mailbox: "INBOX", Entries: []string{"/shared/comment"}, HasMaxSize: true, MaxSize: 1024,
		},
		`tag GETMETADATA (depth INFINITY maxsize 0) INBOX /shared/comment`: {
			Mailbox: "INBOX", Entries: []string{"/shared/comment"}, Depth: command.MetadataDepthInfinity, HasMaxSize: true,
		},
		`tag GETMETADATA INBOX (DEPTH 1) "/private/filters/values"`: {
			Mailbox: "INBOX", Entries: []string{"/private/filters/values"}, Depth: command.MetadataDepthOne,
		},
		`tag GETMETADATA INBOX ("/private/a b" /private/c)`: {
			Mailbox: "INBOX", Entries: []string{"/private/a b", "/private/c"},
		},
	} {
		cmd, err := parseCommand(line)
		require.NoError(t, err, line)
		require.Equal(t, command.Command{Tag: "tag", Payload: expected}, cmd, line)
	}
}

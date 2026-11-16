package tests

import (
	"fmt"
	"testing"

	"github.com/ProtonMail/gluon/imap/command"
	"github.com/stretchr/testify/require"
)

func TestUTF8AcceptMailboxNames(t *testing.T) {
	runManyToOneTestWithAuth(t, defaultServerOptions(t), []int{1, 2}, func(c map[int]*testConnection, _ *testSession) {
		// Without UTF8=ACCEPT a mailbox name is written in modified UTF-7.
		c[1].C(`A001 CREATE "Entw&APw-rfe"`).OK(`A001`)

		c[1].C(`A002 LIST "" "*"`)
		c[1].S(`* LIST (\HasNoChildren \Unmarked) "/" "INBOX"`, `* LIST (\HasNoChildren \Unmarked) "/" "Entw&APw-rfe"`)
		c[1].OK(`A002`)

		// Once it is enabled, names are UTF-8 in both directions.
		c[1].C(`A003 ENABLE UTF8=ACCEPT`).S(`* ENABLED UTF8=ACCEPT`).OK(`A003`)

		c[1].C(`A004 LIST "" "*"`)
		c[1].S(`* LIST (\HasNoChildren \Unmarked) "/" "INBOX"`, `* LIST (\HasNoChildren \Unmarked) "/" "Entwürfe"`)
		c[1].OK(`A004`)

		c[1].C(`A005 CREATE "Größe"`).OK(`A005`)
		c[1].C(`A006 STATUS "Größe" (MESSAGES)`).S(`* STATUS "Größe" (MESSAGES 0)`).OK(`A006`)
		c[1].C(`A007 SELECT "Größe"`).OK(`A007`)
		c[1].C(`A008 RENAME "Größe" "Maße"`).OK(`A008`)

		c[1].C(`A009 LSUB "" "*"`)
		c[1].S(`* LSUB (\Unmarked) "/" "INBOX"`, `* LSUB (\Unmarked) "/" "Entwürfe"`, `* LSUB (\Unmarked) "/" "Maße"`)
		c[1].OK(`A009`)

		c[1].C(`A010 LIST "" "Ma%" RETURN (STATUS (MESSAGES))`)
		c[1].S(`* LIST (\HasNoChildren \Unmarked) "/" "Maße"`, `* STATUS "Maße" (MESSAGES 0)`)
		c[1].OK(`A010`)

		// The other session has not enabled it and still reads and writes modified UTF-7.
		c[2].C(`B001 LIST "" "*"`)
		c[2].S(
			`* LIST (\HasNoChildren \Unmarked) "/" "INBOX"`,
			`* LIST (\HasNoChildren \Unmarked) "/" "Entw&APw-rfe"`,
			`* LIST (\HasNoChildren \Unmarked) "/" "Ma&AN8-e"`,
		)
		c[2].OK(`B001`)

		c[2].C(`B002 SELECT "Ma&AN8-e"`).OK(`B002`)
	})
}

func TestUTF8AcceptAmpersandIsAPlainCharacter(t *testing.T) {
	runManyToOneTestWithAuth(t, defaultServerOptions(t), []int{1, 2}, func(c map[int]*testConnection, _ *testSession) {
		c[1].C(`A001 ENABLE UTF8=ACCEPT`).S(`* ENABLED UTF8=ACCEPT`).OK(`A001`)

		// In UTF-8 an ampersand does not start an encoded part of the name.
		c[1].C(`A002 CREATE "R&D"`).OK(`A002`)
		c[1].C(`A003 LIST "" "R*"`).S(`* LIST (\HasNoChildren \Unmarked) "/" "R&D"`).OK(`A003`)

		c[2].C(`B001 LIST "" "R*"`).S(`* LIST (\HasNoChildren \Unmarked) "/" "R&-D"`).OK(`B001`)
		c[2].C(`B002 SELECT "R&-D"`).OK(`B002`)
	})
}

func TestUTF8AcceptInvalidMailboxNames(t *testing.T) {
	runOneToOneTestWithAuth(t, defaultServerOptions(t), func(c *testConnection, _ *testSession) {
		c.C(`A001 ENABLE UTF8=ACCEPT`).S(`* ENABLED UTF8=ACCEPT`).OK(`A001`)

		for i, name := range []string{
			// Bytes that are not UTF-8.
			"bad\xffname",
			"cut\xc3",
			// Characters RFC 6855 does not allow in a mailbox name: C1 controls and the Unicode line separators.
			"control\u0085",
			"delete\u007f",
			"line separator",
			"paragraph separator",
		} {
			tag := fmt.Sprintf("B%03d", i)

			c.Cf(`%v CREATE "%v"`, tag, name)
			c.Sx(`^` + tag + ` BAD`)

			c.Cf(`%v SELECT "%v"`, tag, name)
			c.Sx(`^` + tag + ` BAD`)

			c.Cf(`%v NOOP`, tag).OK(tag)
		}

		c.C(`A002 LIST "" "*"`).S(`* LIST (\HasNoChildren \Unmarked) "/" "INBOX"`).OK(`A002`)
	})
}

func TestUTF8AcceptSearchWithoutCharset(t *testing.T) {
	runOneToOneTestWithAuth(t, defaultServerOptions(t), func(c *testConnection, _ *testSession) {
		c.doAppend(`INBOX`, buildRFC5322TestLiteral("Subject: café\r\n\r\nbody")).expect("OK")
		c.C(`A001 SELECT INBOX`).OK(`A001`)

		c.C("A002 SEARCH CHARSET UTF-8 SUBJECT \"café\"").S(`* SEARCH 1`).OK(`A002`)

		c.C(`A003 ENABLE UTF8=ACCEPT`).S(`* ENABLED UTF8=ACCEPT`).OK(`A003`)

		// RFC 6855: a client that enabled UTF8=ACCEPT must not name a charset. Search strings are UTF-8.
		c.C("A004 SEARCH CHARSET UTF-8 SUBJECT \"café\"").BAD(`A004`)
		c.C(`A005 UID SEARCH CHARSET US-ASCII ALL`).BAD(`A005`)
		c.C("A006 SEARCH SUBJECT \"café\"").S(`* SEARCH 1`).OK(`A006`)
		c.C(`A007 SEARCH RETURN (COUNT) ALL`).S(`* ESEARCH (TAG "A007") COUNT 1`).OK(`A007`)
	})
}

func TestUTF8AcceptAppend(t *testing.T) {
	runOneToOneTestWithAuth(t, defaultServerOptions(t), func(c *testConnection, _ *testSession) {
		first := buildRFC5322TestLiteral("Subject: Größe\r\n\r\nfirst")
		second := buildRFC5322TestLiteral("Subject: Maße\r\n\r\nsecond")

		c.C(`A001 ENABLE UTF8=ACCEPT`).S(`* ENABLED UTF8=ACCEPT`).OK(`A001`)

		// A message with UTF-8 in its header is sent with the UTF8 data extension of APPEND.
		c.Cf("A002 APPEND INBOX UTF8 (~{%v+}\r\n%v)", len(first), first)
		c.Sx(`^A002 OK \[APPENDUID \d+ 1\]`)

		c.Cf("A003 APPEND INBOX (\\Seen) \"13-Nov-2026 10:00:00 +0000\" UTF8 (~{%v}", len(second))
		c.Sx(`\+.*`)
		c.C(second + ")")
		c.Sx(`^A003 OK \[APPENDUID \d+ 2\]`)

		// It can be one of several messages.
		c.Cf("A004 APPEND INBOX {%v+}\r\n%v utf8 ({%v+}\r\n%v)", len(first), first, len(second), second)
		c.Sx(`^A004 OK \[APPENDUID \d+ 3:4\]`)

		c.C(`A005 SELECT INBOX`).OK(`A005`)
		c.C("A006 SEARCH SUBJECT \"Größe\"").S(`* SEARCH 1 3`).OK(`A006`)
		c.C(`A007 SEARCH SEEN`).S(`* SEARCH 2`).OK(`A007`)

		c.C(`A008 FETCH 2 (BODY.PEEK[HEADER.FIELDS (SUBJECT)])`)
		c.S("* 2 FETCH (BODY[HEADER.FIELDS (SUBJECT)] {18}\r\nSubject: Maße\r\n\r\n)")
		c.OK(`A008`)
	})
}

func TestUTF8AcceptAppendFailures(t *testing.T) {
	runOneToOneTestWithAuth(t, defaultServerOptions(t), func(c *testConnection, _ *testSession) {
		literal := buildRFC5322TestLiteral("Subject: plain\r\n\r\nbody")

		c.C(`A001 ENABLE UTF8=ACCEPT`).S(`* ENABLED UTF8=ACCEPT`).OK(`A001`)

		c.Cf("A002 APPEND INBOX UTF8 ~{%v+}\r\n%v", len(literal), literal).BAD(`A002`)
		c.Cf("A003 APPEND INBOX UTF8 (~{%v+}\r\n%v", len(literal), literal).BAD(`A003`)
		c.Cf("A004 APPEND INBOX UTF7 (~{%v+}\r\n%v)", len(literal), literal).BAD(`A004`)
		c.Cf("A005 APPEND INBOX UTF8 (~{%v+}\r\n%v) extra", len(literal), literal).BAD(`A005`)
		c.C(`A006 APPEND INBOX UTF8 ()`).BAD(`A006`)
		c.Cf("A007 APPEND Missing UTF8 (~{%v+}\r\n%v)", len(literal), literal).NO(`A007`)

		c.C(`A008 STATUS INBOX (MESSAGES)`).S(`* STATUS "INBOX" (MESSAGES 0)`).OK(`A008`)
	})
}

func TestUTF8AcceptEndsWithTheLogin(t *testing.T) {
	runOneToOneTestWithAuth(t, defaultServerOptions(t), func(c *testConnection, _ *testSession) {
		c.C(`A001 ENABLE UTF8=ACCEPT`).S(`* ENABLED UTF8=ACCEPT`).OK(`A001`)
		c.C(`A002 CREATE "Entwürfe"`).OK(`A002`)
		c.C(`A003 LIST "" "E*"`).S(`* LIST (\HasNoChildren \Unmarked) "/" "Entwürfe"`).OK(`A003`)

		// RFC 8437: what was enabled with ENABLE is no longer enabled after UNAUTHENTICATE.
		c.C(`A004 UNAUTHENTICATE`).OK(`A004`)
		c.C(`A005 LOGIN user pass`).OK(`A005`)

		c.C(`A006 LIST "" "E*"`).S(`* LIST (\HasNoChildren \Unmarked) "/" "Entw&APw-rfe"`).OK(`A006`)

		c.C(`A007 ENABLE UTF8=ACCEPT`).S(`* ENABLED UTF8=ACCEPT`).OK(`A007`)
		c.C(`A008 LIST "" "E*"`).S(`* LIST (\HasNoChildren \Unmarked) "/" "Entwürfe"`).OK(`A008`)
	})
}

func TestParserAppendUTF8(t *testing.T) {
	cmd, err := parseCommand(`tag APPEND INBOX (\Seen) UTF8 (~{5}`, `hello)`)
	require.NoError(t, err)
	require.Equal(t, &command.Append{Mailbox: "INBOX", Flags: []string{`\Seen`}, Literal: []byte("hello")}, cmd.Payload)

	cmd, err = parseCommand(`tag APPEND INBOX utf8 ({5+}`, `hello) UTF8 (~{3+}`, `abc)`)
	require.NoError(t, err)
	require.Equal(t, &command.Append{
		Mailbox: "INBOX",
		Literal: []byte("hello"),
		More:    []command.AppendMessage{{Literal: []byte("abc")}},
	}, cmd.Payload)

	for _, lines := range [][]string{
		{`tag APPEND INBOX UTF8 ~{5}`, `hello`},
		{`tag APPEND INBOX UTF8 (~{5}`, `hello`},
		{`tag APPEND INBOX UTF8(~{5}`, `hello)`},
		{`tag APPEND INBOX UTF9 (~{5}`, `hello)`},
	} {
		_, err := parseCommand(lines...)
		require.Error(t, err, lines)
	}
}

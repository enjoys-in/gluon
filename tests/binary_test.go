package tests

import (
	"fmt"
	"testing"

	"github.com/ProtonMail/gluon/imap/command"
	"github.com/stretchr/testify/require"
)

// The parts of binaryTestMessage: plain text, a base64 attachment that holds a zero byte, quoted-printable text
// and a part in an encoding the server does not know.
const binaryTestMessage = "From: sender@pm.me\r\n" +
	"To: user@pm.me\r\n" +
	"Date: Mon, 01 Jan 2024 10:00:00 +0000\r\n" +
	"Subject: binary\r\n" +
	"Content-Type: multipart/mixed; boundary=part\r\n" +
	"\r\n" +
	"--part\r\n" +
	"Content-Type: text/plain\r\n" +
	"\r\n" +
	"Hello\r\n" +
	"--part\r\n" +
	"Content-Type: application/octet-stream\r\n" +
	"Content-Transfer-Encoding: BASE64\r\n" +
	"\r\n" +
	"YQBi\r\n" +
	"/2M=\r\n" +
	"--part\r\n" +
	"Content-Type: text/plain; charset=iso-8859-1\r\n" +
	"Content-Transfer-Encoding: quoted-printable\r\n" +
	"\r\n" +
	"Caf=E9 au =\r\n" +
	"lait\r\n" +
	"--part\r\n" +
	"Content-Type: application/x-custom\r\n" +
	"Content-Transfer-Encoding: x-uuencode\r\n" +
	"\r\n" +
	"begin 644 file\r\n" +
	"--part--\r\n"

// expectBytes expects the next response to be exactly the given bytes. The patterns the other helpers use cannot
// hold bytes that are not UTF-8.
func expectBytes(c *testConnection, want string) *testConnection {
	require.Equal(c.tb, want+"\r\n", string(c.read()))

	return c
}

func appendBinaryTestMessage(c *testConnection) {
	c.doAppend("INBOX", binaryTestMessage).expect("OK")
	c.C(`A000 SELECT INBOX`).OK(`A000`)
}

func TestBinaryFetch(t *testing.T) {
	runOneToOneTestWithAuth(t, defaultServerOptions(t), func(c *testConnection, _ *testSession) {
		appendBinaryTestMessage(c)

		// A part without an encoding is sent as it is.
		c.C(`A001 FETCH 1 (BINARY.PEEK[1])`).S("* 1 FETCH (BINARY[1] {5}\r\nHello)").OK(`A001`)

		// The base64 part is decoded. It holds a zero byte, so it is sent as a binary literal.
		c.C(`A002 FETCH 1 (BINARY.PEEK[2])`)
		expectBytes(c, "* 1 FETCH (BINARY[2] ~{5}\r\na\x00b\xffc)").OK(`A002`)

		c.C(`A003 FETCH 1 (BINARY.PEEK[3])`)
		expectBytes(c, "* 1 FETCH (BINARY[3] {12}\r\nCaf\xe9 au lait)").OK(`A003`)

		c.C(`A004 UID FETCH 1 (binary.peek[2] BINARY.PEEK[1])`)
		expectBytes(c, "* 1 FETCH (BINARY[2] ~{5}\r\na\x00b\xffc BINARY[1] {5}\r\nHello UID 1)").OK(`A004`)
	})
}

func TestBinaryFetchSize(t *testing.T) {
	runOneToOneTestWithAuth(t, defaultServerOptions(t), func(c *testConnection, _ *testSession) {
		appendBinaryTestMessage(c)

		// The size is the size of the decoded part.
		c.C(`A001 FETCH 1 (BINARY.SIZE[1] BINARY.SIZE[2] BINARY.SIZE[3])`)
		c.S(`* 1 FETCH (BINARY.SIZE[1] 5 BINARY.SIZE[2] 5 BINARY.SIZE[3] 12)`)
		c.OK(`A001`)

		// Asking for a size does not mark the message as read.
		c.C(`A002 FETCH 1 (FLAGS)`).S(`* 1 FETCH (FLAGS (\Recent))`).OK(`A002`)
	})
}

func TestBinaryFetchPartial(t *testing.T) {
	runOneToOneTestWithAuth(t, defaultServerOptions(t), func(c *testConnection, _ *testSession) {
		appendBinaryTestMessage(c)

		// The range counts in the decoded part.
		c.C(`A001 FETCH 1 (BINARY.PEEK[2]<1.2>)`).S("* 1 FETCH (BINARY[2]<1> ~{2}\r\n\x00b)").OK(`A001`)
		c.C(`A002 FETCH 1 (BINARY.PEEK[2]<3.100>)`)
		expectBytes(c, "* 1 FETCH (BINARY[2]<3> {2}\r\n\xffc)").OK(`A002`)
		c.C(`A003 FETCH 1 (BINARY.PEEK[1]<2.2>)`).S("* 1 FETCH (BINARY[1]<2> {2}\r\nll)").OK(`A003`)
		c.C(`A004 FETCH 1 (BINARY.PEEK[3]<0.4>)`)
		expectBytes(c, "* 1 FETCH (BINARY[3]<0> {4}\r\nCaf\xe9)").OK(`A004`)
	})
}

func TestBinaryFetchMarksTheMessageAsRead(t *testing.T) {
	runOneToOneTestWithAuth(t, defaultServerOptions(t), func(c *testConnection, _ *testSession) {
		appendBinaryTestMessage(c)

		c.C(`A001 FETCH 1 (BINARY.PEEK[1])`).S("* 1 FETCH (BINARY[1] {5}\r\nHello)").OK(`A001`)
		c.C(`A002 FETCH 1 (BINARY[1])`).S("* 1 FETCH (BINARY[1] {5}\r\nHello FLAGS (\\Recent \\Seen))").OK(`A002`)
	})
}

func TestBinaryFetchOfTheWholeMessage(t *testing.T) {
	runOneToOneTestWithAuth(t, defaultServerOptions(t), func(c *testConnection, _ *testSession) {
		appendBinaryTestMessage(c)

		// The message itself has no encoding to remove.
		c.C(`A001 FETCH 1 (BINARY.SIZE[] RFC822.SIZE)`)
		c.Sx(`^\* 1 FETCH \(BINARY\.SIZE\[\] (\d+) RFC822\.SIZE (\d+)\)`)
		c.OK(`A001`)

		c.C(`A002 FETCH 1 (BINARY.PEEK[]<0.14>)`).S("* 1 FETCH (BINARY[]<0> {14}\r\nX-Pm-Gluon-Id:)").OK(`A002`)
	})
}

func TestBinaryFetchFailures(t *testing.T) {
	runOneToOneTestWithAuth(t, defaultServerOptions(t), func(c *testConnection, _ *testSession) {
		appendBinaryTestMessage(c)

		// The fourth part has an encoding the server cannot remove.
		c.C(`A001 FETCH 1 (BINARY.PEEK[4])`)
		c.Sx(`^A001 NO \[UNKNOWN-CTE\]`)

		c.C(`A002 FETCH 1 (BINARY.SIZE[4])`)
		c.Sx(`^A002 NO \[UNKNOWN-CTE\]`)

		c.C(`A003 FETCH 1 (BINARY.PEEK[9])`).NO(`A003`)

		for i, line := range []string{
			`FETCH 1 (BINARY)`,
			`FETCH 1 (BINARY[1.TEXT])`,
			`FETCH 1 (BINARY[HEADER])`,
			`FETCH 1 (BINARY[1.])`,
			`FETCH 1 (BINARY[1)`,
			`FETCH 1 (BINARY.FOO[1])`,
			`FETCH 1 (BINARY.SIZE[1]<0.5>)`,
			`FETCH 1 (BINARY.PEEK[1]<0.0>)`,
			`FETCH 1 (BINARY.PEEK[0])`,
		} {
			tag := fmt.Sprintf("B%03d", i)

			c.Cf(`%v %v`, tag, line)
			c.Sx(`^` + tag + ` BAD`)
		}

		// Nothing above marked the message as read.
		c.C(`A004 FETCH 1 (FLAGS)`).S(`* 1 FETCH (FLAGS (\Recent))`).OK(`A004`)

		c.C(`A005 CAPABILITY`)
		c.Sx(`\* CAPABILITY .* BINARY `)
		c.OK(`A005`)
	})
}

func TestBinaryAppend(t *testing.T) {
	runOneToOneTestWithAuth(t, defaultServerOptions(t), func(c *testConnection, _ *testSession) {
		binary := buildRFC5322TestLiteral("Content-Transfer-Encoding: binary\r\n\r\nzero\x00byte\r\n")
		plain := buildRFC5322TestLiteral("To: 2@pm.me")

		// A message with binary content is sent as a binary literal.
		c.Cf("A001 APPEND INBOX (\\Seen) ~{%v+}\r\n%v", len(binary), binary).OK(`A001`)

		c.Cf("A002 APPEND INBOX ~{%v}", len(plain))
		c.Sx(`\+.*`)
		c.C(plain).OK(`A002`)

		// It works for every message of an APPEND with several of them.
		c.Cf("A003 APPEND INBOX ~{%v+}\r\n%v \"13-Nov-2026 10:00:00 +0000\" ~{%v+}\r\n%v", len(binary), binary, len(plain), plain)
		c.Sx(`^A003 OK \[APPENDUID \d+ 3:4\]`)

		c.C(`A004 SELECT INBOX`).OK(`A004`)
		c.C(`A005 FETCH 1 (BINARY.PEEK[1])`).S("* 1 FETCH (BINARY[1] ~{11}\r\nzero\x00byte\r\n)").OK(`A005`)
		c.C(`A006 FETCH 3 (BINARY.SIZE[1])`).S(`* 3 FETCH (BINARY.SIZE[1] 11)`).OK(`A006`)

		c.C(`A007 APPEND INBOX ~hello`).BAD(`A007`)
	})
}

func TestParserBinary(t *testing.T) {
	cmd, err := parseCommand(`tag FETCH 1 (BINARY[1.2.3]<10.20> binary.peek[] BINARY.SIZE[4])`)
	require.NoError(t, err)
	require.Equal(t, []command.FetchAttribute{
		&command.FetchAttributeBinary{Part: []int{1, 2, 3}, Partial: &command.BodySectionPartial{Offset: 10, Count: 20}},
		&command.FetchAttributeBinary{Peek: true},
		&command.FetchAttributeBinarySize{Part: []int{4}},
	}, cmd.Payload.(*command.Fetch).Attributes)

	require.Equal(t, "BINARY[1.2.3]", cmd.Payload.(*command.Fetch).Attributes[0].String())
	require.Equal(t, "BINARY.PEEK[]", cmd.Payload.(*command.Fetch).Attributes[1].String())
	require.Equal(t, "BINARY.SIZE[4]", cmd.Payload.(*command.Fetch).Attributes[2].String())

	cmd, err = parseCommand(`tag APPEND INBOX (\Seen) ~{5}`, `hello`)
	require.NoError(t, err)
	require.Equal(t, &command.Append{Mailbox: "INBOX", Flags: []string{`\Seen`}, Literal: []byte("hello")}, cmd.Payload)
}

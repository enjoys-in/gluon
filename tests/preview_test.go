package tests

import (
	"strings"
	"testing"

	"github.com/ProtonMail/gluon/imap/command"
	"github.com/stretchr/testify/require"
)

// appendPreviewMessage appends a message with the given headers and body to INBOX.
func appendPreviewMessage(c *testConnection, headers, body string) {
	c.doAppend("INBOX", buildRFC5322TestLiteral("Subject: preview\r\n"+headers+"\r\n\r\n"+body)).expect("OK")
}

func TestPreviewOfPlainText(t *testing.T) {
	runOneToOneTestWithAuth(t, defaultServerOptions(t), func(c *testConnection, _ *testSession) {
		appendPreviewMessage(c, "Content-Type: text/plain", "Hello   there,\r\n\r\n\tthis is the\r\nfirst message.\r\n")
		c.C(`A000 SELECT INBOX`).OK(`A000`)

		// Line breaks and runs of spaces become one space.
		c.C(`A001 FETCH 1 PREVIEW`).S(`* 1 FETCH (PREVIEW "Hello there, this is the first message.")`).OK(`A001`)
		c.C(`A002 FETCH 1 (PREVIEW)`).S(`* 1 FETCH (PREVIEW "Hello there, this is the first message.")`).OK(`A002`)
		c.C(`A003 FETCH 1 (UID PREVIEW (LAZY) RFC822.SIZE)`).Sx(`^\* 1 FETCH \(UID 1 PREVIEW "Hello there, this is the first message\." RFC822\.SIZE \d+\)`).OK(`A003`)
		c.C(`A004 FETCH 1 PREVIEW (LAZY)`).S(`* 1 FETCH (PREVIEW "Hello there, this is the first message.")`).OK(`A004`)
		c.C(`A005 UID FETCH 1 preview (lazy)`).S(`* 1 FETCH (PREVIEW "Hello there, this is the first message." UID 1)`).OK(`A005`)

		// Asking for the preview does not mark the message as read.
		c.C(`A006 FETCH 1 (FLAGS)`).S(`* 1 FETCH (FLAGS (\Recent))`).OK(`A006`)
	})
}

func TestPreviewIsCutAfter200Characters(t *testing.T) {
	runOneToOneTestWithAuth(t, defaultServerOptions(t), func(c *testConnection, _ *testSession) {
		appendPreviewMessage(c, "Content-Type: text/plain", strings.Repeat("0123456789", 30))
		appendPreviewMessage(c, "Content-Type: text/plain; charset=utf-8", strings.Repeat("é", 300))
		appendPreviewMessage(c, "Content-Type: text/plain", strings.Repeat("a", 200))
		c.C(`A000 SELECT INBOX`).OK(`A000`)

		c.C(`A001 FETCH 1 PREVIEW`).S(`* 1 FETCH (PREVIEW "` + strings.Repeat("0123456789", 20) + `")`).OK(`A001`)

		// The limit counts characters, not bytes.
		c.C(`A002 FETCH 2 PREVIEW`).S("* 2 FETCH (PREVIEW {400}\r\n" + strings.Repeat("é", 200) + ")").OK(`A002`)

		c.C(`A003 FETCH 3 PREVIEW`).S(`* 3 FETCH (PREVIEW "` + strings.Repeat("a", 200) + `")`).OK(`A003`)
	})
}

func TestPreviewDecodesTheBody(t *testing.T) {
	runOneToOneTestWithAuth(t, defaultServerOptions(t), func(c *testConnection, _ *testSession) {
		// "Hello from base64" in base64, split over two lines.
		appendPreviewMessage(c, "Content-Type: text/plain\r\nContent-Transfer-Encoding: BASE64", "SGVsbG8gZnJv\r\nbSBiYXNlNjQ=\r\n")
		appendPreviewMessage(c, "Content-Type: text/plain; charset=iso-8859-1\r\nContent-Transfer-Encoding: quoted-printable", "Caf=E9 au lait=\r\n, s'il vous pla=EEt")
		appendPreviewMessage(c, "Content-Type: text/plain", `She said "hi" and left a \ behind`)
		c.C(`A000 SELECT INBOX`).OK(`A000`)

		c.C(`A001 FETCH 1 PREVIEW`).S(`* 1 FETCH (PREVIEW "Hello from base64")`).OK(`A001`)

		// Text that is not plain ASCII is converted to UTF-8 and sent as a literal.
		c.C(`A002 FETCH 2 PREVIEW`).S("* 2 FETCH (PREVIEW {31}\r\nCafé au lait, s'il vous plaît)").OK(`A002`)

		c.C(`A003 FETCH 3 PREVIEW`).S(`* 3 FETCH (PREVIEW "She said \"hi\" and left a \\ behind")`).OK(`A003`)
	})
}

func TestPreviewPicksTheTextPart(t *testing.T) {
	runOneToOneTestWithAuth(t, defaultServerOptions(t), func(c *testConnection, _ *testSession) {
		// Plain text is preferred to HTML, wherever it is in the message.
		appendPreviewMessage(c, "Content-Type: multipart/alternative; boundary=alt",
			"--alt\r\nContent-Type: text/html\r\n\r\n<p>The <b>HTML</b> version</p>\r\n"+
				"--alt\r\nContent-Type: text/plain\r\n\r\nThe plain version\r\n"+
				"--alt--\r\n")

		// A text file that is attached is not the text of the message.
		appendPreviewMessage(c, "Content-Type: multipart/mixed; boundary=mix",
			"--mix\r\nContent-Type: text/plain\r\nContent-Disposition: attachment; filename=notes.txt\r\n\r\nattached notes\r\n"+
				"--mix\r\nContent-Type: multipart/alternative; boundary=alt\r\n\r\n"+
				"--alt\r\nContent-Type: text/plain\r\n\r\nSee the attached notes.\r\n"+
				"--alt--\r\n"+
				"--mix--\r\n")

		// Without plain text the HTML part is used: no tags, no style sheet, no script, entities decoded.
		appendPreviewMessage(c, "Content-Type: text/html",
			"<html><head><title>Offer</title><style>p { color: red }</style></head>\r\n"+
				"<body><!-- tracking --><script>track()</script><p>Fish &amp; chips<br>for &lt;2&gt; people</p></body></html>")

		// A message with no text has an empty preview.
		appendPreviewMessage(c, "Content-Type: application/octet-stream\r\nContent-Transfer-Encoding: base64", "AAECAwQ=")

		c.C(`A000 SELECT INBOX`).OK(`A000`)

		c.C(`A001 FETCH 1:4 PREVIEW`)
		c.S(
			`* 1 FETCH (PREVIEW "The plain version")`,
			`* 2 FETCH (PREVIEW "See the attached notes.")`,
			`* 3 FETCH (PREVIEW "Fish & chips for <2> people")`,
			`* 4 FETCH (PREVIEW "")`,
		)
		c.OK(`A001`)
	})
}

func TestPreviewFailures(t *testing.T) {
	runOneToOneTest(t, defaultServerOptions(t), func(c *testConnection, _ *testSession) {
		c.C(`A001 FETCH 1 PREVIEW`).NO(`A001`)

		c.C(`A002 LOGIN user pass`).OK(`A002`)
		c.C(`A003 FETCH 1 PREVIEW`).NO(`A003`)

		appendPreviewMessage(c, "Content-Type: text/plain", "text")
		c.C(`A004 SELECT INBOX`).OK(`A004`)

		c.C(`A005 FETCH 1 (PREVIEW (EAGER))`).BAD(`A005`)
		c.C(`A006 FETCH 1 (PREVIEW (LAZY EAGER))`).BAD(`A006`)
		c.C(`A007 FETCH 1 (PREVIEW ())`).BAD(`A007`)
		c.C(`A008 FETCH 1 (UID (LAZY))`).BAD(`A008`)
		c.C(`A009 FETCH 1 PREVIEW LAZY`).BAD(`A009`)
		c.C(`A010 FETCH 1 PREVIEW (LAZY`).BAD(`A010`)
		c.C(`A011 FETCH 1 PREVIEWS`).BAD(`A011`)
		c.C(`A012 FETCH 5 PREVIEW`).BAD(`A012`)

		c.C(`A013 FETCH 1 PREVIEW`).S(`* 1 FETCH (PREVIEW "text")`).OK(`A013`)

		c.C(`A014 CAPABILITY`)
		c.Sx(`\* CAPABILITY .* PREVIEW `)
		c.OK(`A014`)
	})
}

func TestParserFetchPreview(t *testing.T) {
	cmd, err := parseCommand(`tag FETCH 1 (UID PREVIEW (LAZY) FLAGS PREVIEW)`)
	require.NoError(t, err)
	require.Equal(t, []command.FetchAttribute{
		&command.FetchAttributeUID{},
		&command.FetchAttributePreview{Lazy: true},
		&command.FetchAttributeFlags{},
		&command.FetchAttributePreview{},
	}, cmd.Payload.(*command.Fetch).Attributes)

	cmd, err = parseCommand(`tag FETCH 1 PREVIEW (lazy)`)
	require.NoError(t, err)
	require.Equal(t, []command.FetchAttribute{&command.FetchAttributePreview{Lazy: true}}, cmd.Payload.(*command.Fetch).Attributes)
	require.Equal(t, "PREVIEW (LAZY)", cmd.Payload.(*command.Fetch).Attributes[0].String())
}

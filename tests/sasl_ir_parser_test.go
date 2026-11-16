package tests

import (
	"encoding/base64"
	"testing"

	"github.com/ProtonMail/gluon/imap/command"
	"github.com/ProtonMail/gluon/rfcparser"
	"github.com/stretchr/testify/require"
)

func TestParserAuthenticateInitialResponse(t *testing.T) {
	for _, data := range []*command.Authenticate{
		{UserID: "user1@example.com", Password: "pass"},
		{UserID: "user1@example.com", Password: ""},
		{UserID: "", Password: ""},
	} {
		continuations := 0

		authString := base64.StdEncoding.EncodeToString([]byte("\x00" + data.UserID + "\x00" + data.Password))
		p := newCommandParser([]string{`A001 AUTHENTICATE PLAIN ` + authString}, countContinuations(&continuations))

		cmd, err := p.Parse()
		require.NoError(t, err)
		require.Zero(t, continuations)
		require.Equal(t, command.Command{Tag: "A001", Payload: data}, cmd)
		require.Equal(t, "authenticate", p.LastParsedCommand())
	}
}

func TestParserAuthenticateInitialResponseFailures(t *testing.T) {
	valid := base64.StdEncoding.EncodeToString([]byte("\x00user\x00pass"))

	for _, test := range []struct {
		line, message string
	}{
		{line: `A001 AUTHENTICATE PLAIN =`, message: "invalid authentication data"},
		{line: `A001 AUTHENTICATE PLAIN not-base64`, message: "invalid base64 content"},
		{line: `A001 AUTHENTICATE PLAIN ` + base64.StdEncoding.EncodeToString([]byte("user pass")), message: "invalid authentication data"},
		{line: `A001 AUTHENTICATE PLAIN ` + valid + ` extra`, message: "expected CR"},
		{line: `A001 AUTHENTICATE LOGIN ` + valid, message: "unsupported authentication mechanism"},
	} {
		continuations := 0

		_, err := newCommandParser([]string{test.line}, countContinuations(&continuations)).Parse()

		var parserError *rfcparser.Error

		require.ErrorAs(t, err, &parserError, test.line)
		require.Equal(t, test.message, parserError.Message, test.line)
		require.Zero(t, continuations, test.line)
	}
}

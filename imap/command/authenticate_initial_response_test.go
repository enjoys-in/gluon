package command

import (
	"bytes"
	"encoding/base64"
	"testing"

	"github.com/ProtonMail/gluon/rfcparser"
	"github.com/stretchr/testify/require"
)

func TestParser_AuthenticateInitialResponse(t *testing.T) {
	for _, data := range []*Authenticate{
		{UserID: "user1@example.com", Password: "pass"},
		{UserID: "user1@example.com", Password: ""},
		{UserID: "", Password: ""},
	} {
		var continued bool

		authString := base64.StdEncoding.EncodeToString([]byte("\x00" + data.UserID + "\x00" + data.Password))
		s := rfcparser.NewScanner(bytes.NewReader(toIMAPLine(`A001 AUTHENTICATE PLAIN ` + authString)))
		p := NewParser(s, WithLiteralContinuationCallback(continuationChecker(&continued)))

		cmd, err := p.Parse()
		require.NoError(t, err)
		require.False(t, continued)
		require.Equal(t, Command{Tag: "A001", Payload: data}, cmd)
		require.Equal(t, "authenticate", p.LastParsedCommand())
	}
}

func TestParser_AuthenticateInitialResponseFailures(t *testing.T) {
	valid := base64.StdEncoding.EncodeToString([]byte("\x00user\x00pass"))

	for _, test := range []struct {
		line, message string
	}{
		{line: `A001 AUTHENTICATE PLAIN =`, message: messageInvalidAuthenticationData},
		{line: `A001 AUTHENTICATE PLAIN not-base64`, message: messageInvalidBase64Content},
		{line: `A001 AUTHENTICATE PLAIN ` + base64.StdEncoding.EncodeToString([]byte("user pass")), message: messageInvalidAuthenticationData},
		{line: `A001 AUTHENTICATE PLAIN ` + valid + ` extra`, message: "expected CR"},
		{line: `A001 AUTHENTICATE LOGIN ` + valid, message: messageUnsupportedAuthenticationMechanism},
	} {
		var continued bool

		s := rfcparser.NewScanner(bytes.NewReader(toIMAPLine(test.line)))
		p := NewParser(s, WithLiteralContinuationCallback(continuationChecker(&continued)))

		_, err := p.Parse()

		var parserError *rfcparser.Error

		require.ErrorAs(t, err, &parserError, test.line)
		require.Equal(t, test.message, parserError.Message, test.line)
		require.False(t, continued, test.line)
	}
}

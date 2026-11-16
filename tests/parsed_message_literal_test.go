package tests

import (
	"bytes"
	"testing"

	"github.com/ProtonMail/gluon/imap"

	"github.com/stretchr/testify/require"
)

func TestNewParsedMessage_DoesNotChangeTheLiteral(t *testing.T) {
	for _, literal := range []string{
		"From: a@pm.me\r\nTo: b@pm.me\r\nSubject: folded\r\n subject\r\n\r\nbody",
		"From: a@pm.me\r\nContent-Type: multipart/mixed; boundary=outer\r\n\r\n" +
			"--outer\r\nContent-Type: text/plain\r\n\r\nfirst\r\n" +
			"--outer\r\nContent-Type: message/rfc822\r\n\r\n" +
			"From: c@pm.me\r\nSubject: embedded\r\n\r\ninner body\r\n" +
			"--outer--\r\n",
	} {
		original := []byte(literal)
		parsed := bytes.Clone(original)

		message, err := imap.NewParsedMessage(parsed)
		require.NoError(t, err)
		require.NotEmpty(t, message.Structure)
		require.NotEmpty(t, message.Envelope)
		require.Equal(t, original, parsed)
	}
}

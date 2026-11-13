package tests

import (
	"testing"

	"github.com/ProtonMail/gluon/imap/command"
	"github.com/stretchr/testify/require"
)

func TestParserUnauthenticate(t *testing.T) {
	p := newCommandParser([]string{`tag UNAUTHENTICATE`})

	cmd, err := p.Parse()
	require.NoError(t, err)
	require.Equal(t, command.Command{Tag: "tag", Payload: &command.Unauthenticate{}}, cmd)
	require.Equal(t, "unauthenticate", p.LastParsedCommand())
	require.Equal(t, "tag", p.LastParsedTag())
}

func TestParserUnauthenticateRejectsArguments(t *testing.T) {
	cmd, err := parseCommand(`tag UNAUTHENTICATE now`)
	require.Error(t, err)
	require.Equal(t, command.Command{Tag: "tag"}, cmd)
}

package command

import (
	"bytes"
	"testing"

	"github.com/ProtonMail/gluon/rfcparser"
	"github.com/stretchr/testify/require"
)

func TestParser_UnauthenticateCommand(t *testing.T) {
	input := toIMAPLine(`tag UNAUTHENTICATE`)
	s := rfcparser.NewScanner(bytes.NewReader(input))
	p := NewParser(s)

	expected := Command{Tag: "tag", Payload: &Unauthenticate{}}

	cmd, err := p.Parse()
	require.NoError(t, err)
	require.Equal(t, expected, cmd)
	require.Equal(t, "unauthenticate", p.LastParsedCommand())
	require.Equal(t, "tag", p.LastParsedTag())
}

func TestParser_UnauthenticateCommandRejectsArguments(t *testing.T) {
	p := NewParser(rfcparser.NewScanner(bytes.NewReader(toIMAPLine(`tag UNAUTHENTICATE now`))))

	cmd, err := p.Parse()
	require.Error(t, err)
	require.Equal(t, Command{Tag: "tag"}, cmd)
}

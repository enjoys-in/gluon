package command

import (
	"bytes"
	"testing"

	"github.com/ProtonMail/gluon/rfcparser"
	"github.com/stretchr/testify/require"
)

func TestParser_NamespaceCommand(t *testing.T) {
	input := toIMAPLine(`tag NAMESPACE`)
	s := rfcparser.NewScanner(bytes.NewReader(input))
	p := NewParser(s)

	expected := Command{Tag: "tag", Payload: &Namespace{}}

	cmd, err := p.Parse()
	require.NoError(t, err)
	require.Equal(t, expected, cmd)
	require.Equal(t, "namespace", p.LastParsedCommand())
	require.Equal(t, "tag", p.LastParsedTag())
}

func TestParser_NamespaceCommandLowerCase(t *testing.T) {
	input := toIMAPLine(`tag namespace`)
	s := rfcparser.NewScanner(bytes.NewReader(input))
	p := NewParser(s)

	cmd, err := p.Parse()
	require.NoError(t, err)
	require.Equal(t, Command{Tag: "tag", Payload: &Namespace{}}, cmd)
}

func TestParser_NamespaceCommandRejectsArguments(t *testing.T) {
	input := toIMAPLine(`tag NAMESPACE INBOX`)
	s := rfcparser.NewScanner(bytes.NewReader(input))
	p := NewParser(s)

	_, err := p.Parse()
	require.Error(t, err)
	require.Equal(t, "namespace", p.LastParsedCommand())
	require.Equal(t, "tag", p.LastParsedTag())
}

func TestParser_NamespaceCommandRejectsTrailingSpace(t *testing.T) {
	input := toIMAPLine(`tag NAMESPACE `)
	s := rfcparser.NewScanner(bytes.NewReader(input))
	p := NewParser(s)

	_, err := p.Parse()
	require.Error(t, err)
}

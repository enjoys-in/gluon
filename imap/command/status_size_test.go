package command

import (
	"bytes"
	"testing"

	"github.com/ProtonMail/gluon/rfcparser"
	"github.com/stretchr/testify/require"
)

func TestParser_StatusCommandSize(t *testing.T) {
	input := toIMAPLine(`tag STATUS INBOX (MESSAGES size)`)
	s := rfcparser.NewScanner(bytes.NewReader(input))
	p := NewParser(s)

	expected := Command{Tag: "tag", Payload: &Status{
		Mailbox:    "INBOX",
		Attributes: []StatusAttribute{StatusAttributeMessages, StatusAttributeSize},
	}}

	cmd, err := p.Parse()
	require.NoError(t, err)
	require.Equal(t, expected, cmd)
	require.Equal(t, "SIZE", StatusAttributeSize.String())
}

func TestParser_StatusCommandUnknownAttribute(t *testing.T) {
	p := NewParser(rfcparser.NewScanner(bytes.NewReader(toIMAPLine(`tag STATUS INBOX (SIZES)`))))

	_, err := p.Parse()
	require.Error(t, err)
}

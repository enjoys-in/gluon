package tests

import (
	"testing"

	"github.com/ProtonMail/gluon/imap/command"
	"github.com/stretchr/testify/require"
)

func TestParserStatusSize(t *testing.T) {
	expected := command.Command{Tag: "tag", Payload: &command.Status{
		Mailbox:    "INBOX",
		Attributes: []command.StatusAttribute{command.StatusAttributeMessages, command.StatusAttributeSize},
	}}

	cmd, err := parseCommand(`tag STATUS INBOX (MESSAGES size)`)
	require.NoError(t, err)
	require.Equal(t, expected, cmd)
	require.Equal(t, "SIZE", command.StatusAttributeSize.String())
}

func TestParserStatusUnknownAttribute(t *testing.T) {
	_, err := parseCommand(`tag STATUS INBOX (SIZES)`)
	require.Error(t, err)
}

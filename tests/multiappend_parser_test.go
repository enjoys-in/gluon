package tests

import (
	"testing"
	"time"

	"github.com/ProtonMail/gluon/imap/command"
	"github.com/stretchr/testify/require"
)

func TestParserAppendWithSeveralMessages(t *testing.T) {
	cmd, err := parseCommand(
		`A003 APPEND saved-messages (\Seen) {5}`,
		`first (\Flagged \Draft) "15-Nov-1984 13:37:01 +0730" {6}`,
		`second {5}`,
		`third`,
	)
	require.NoError(t, err)

	expected := &command.Append{
		Mailbox: "saved-messages",
		Flags:   []string{`\Seen`},
		Literal: []byte("first"),
		More: []command.AppendMessage{
			{
				Flags:    []string{`\Flagged`, `\Draft`},
				DateTime: time.Date(1984, time.November, 15, 13, 37, 1, 0, time.FixedZone("zone", 7*3600+30*60)),
				Literal:  []byte("second"),
			},
			{Literal: []byte("third")},
		},
	}

	require.Equal(t, command.Command{Tag: "A003", Payload: expected}, cmd)
	require.Equal(t, []command.AppendMessage{
		{Flags: []string{`\Seen`}, Literal: []byte("first")},
		expected.More[0],
		expected.More[1],
	}, expected.Messages())
}

func TestParserAppendWithSeveralMessagesAsksForEachLiteral(t *testing.T) {
	continuations := 0

	cmd, err := newCommandParser([]string{`A003 APPEND INBOX {5}`, `first {6}`, `second`}, countContinuations(&continuations)).Parse()
	require.NoError(t, err)
	require.Equal(t, 2, continuations)
	require.Len(t, cmd.Payload.(*command.Append).Messages(), 2)
}

func TestParserAppendWithSeveralNonSynchronizingLiterals(t *testing.T) {
	continuations := 0

	cmd, err := newCommandParser([]string{`A003 APPEND INBOX {5+}`, `first {6+}`, `second`}, countContinuations(&continuations)).Parse()
	require.NoError(t, err)
	require.Zero(t, continuations)
	require.Equal(t, []byte("second"), cmd.Payload.(*command.Append).More[0].Literal)
}

func TestParserAppendWithOneMessageHasNoMore(t *testing.T) {
	cmd, err := parseCommand(`A003 APPEND INBOX {5}`, `first`)
	require.NoError(t, err)
	require.Nil(t, cmd.Payload.(*command.Append).More)
	require.Len(t, cmd.Payload.(*command.Append).Messages(), 1)
}

func TestParserAppendWithSeveralMessagesFailures(t *testing.T) {
	for _, lines := range [][]string{
		{`A003 APPEND INBOX {5}`, `first `},
		{`A003 APPEND INBOX {5}`, `first (\Seen)`},
		{`A003 APPEND INBOX {5}`, `first (\Seen) `},
		{`A003 APPEND INBOX {5}`, `first "15-Nov-1984 13:37:01 +0730"`},
		{`A003 APPEND INBOX {5}`, `first second`},
		{`A003 APPEND INBOX {5}`, `first{6}`, `second`},
		{`A003 APPEND INBOX {5}`, `first (\Recent) {6}`, `second`},
	} {
		_, err := parseCommand(lines...)
		require.Error(t, err, lines)
	}
}

package tests

import (
	"strings"
	"testing"

	"github.com/ProtonMail/gluon/imap/command"

	"github.com/ProtonMail/gluon/rfcparser"
	"github.com/stretchr/testify/require"
)

func newLiteralTestParser(input string) (*command.Parser, *int) {
	continuations := new(int)

	p := command.NewParser(
		rfcparser.NewScanner(strings.NewReader(input)),
		command.WithLiteralContinuationCallback(func(string) error {
			*continuations++
			return nil
		}),
	)

	return p, continuations
}

func TestParser_NonSynchronizingLiteralNeedsNoContinuation(t *testing.T) {
	p, continuations := newLiteralTestParser("tag LOGIN {4+}\r\nuser {4+}\r\npass\r\n")

	cmd, err := p.Parse()
	require.NoError(t, err)
	require.Equal(t, command.Command{Tag: "tag", Payload: &command.Login{UserID: "user", Password: "pass"}}, cmd)
	require.Zero(t, *continuations)
}

func TestParser_SynchronizingLiteralStillAsksForContinuation(t *testing.T) {
	p, continuations := newLiteralTestParser("tag LOGIN {4}\r\nuser {4}\r\npass\r\n")

	cmd, err := p.Parse()
	require.NoError(t, err)
	require.Equal(t, command.Command{Tag: "tag", Payload: &command.Login{UserID: "user", Password: "pass"}}, cmd)
	require.Equal(t, 2, *continuations)
}

func TestParser_MixedLiterals(t *testing.T) {
	p, continuations := newLiteralTestParser("tag LOGIN {4}\r\nuser {4+}\r\npass\r\n")

	cmd, err := p.Parse()
	require.NoError(t, err)
	require.Equal(t, command.Command{Tag: "tag", Payload: &command.Login{UserID: "user", Password: "pass"}}, cmd)
	require.Equal(t, 1, *continuations)
}

func TestParser_NonSynchronizingLiteralInAppend(t *testing.T) {
	p, continuations := newLiteralTestParser("tag APPEND INBOX {11+}\r\nTo: a@b.c\r\n\r\n")

	cmd, err := p.Parse()
	require.NoError(t, err)

	appendCmd, ok := cmd.Payload.(*command.Append)
	require.True(t, ok)
	require.Equal(t, "INBOX", appendCmd.Mailbox)
	require.Equal(t, []byte("To: a@b.c\r\n"), appendCmd.Literal)
	require.Zero(t, *continuations)
}

func TestParser_NonSynchronizingLiteralKeepsBinaryData(t *testing.T) {
	p, _ := newLiteralTestParser("tag LOGIN user {6+}\r\na\r\nb{1\r\n")

	cmd, err := p.Parse()
	require.NoError(t, err)
	require.Equal(t, command.Command{Tag: "tag", Payload: &command.Login{UserID: "user", Password: "a\r\nb{1"}}, cmd)
}

func TestParser_ListMailboxAsNonSynchronizingLiteral(t *testing.T) {
	p, continuations := newLiteralTestParser("tag LIST \"\" {5+}\r\nINBOX\r\n")

	cmd, err := p.Parse()
	require.NoError(t, err)
	require.Equal(t, command.Command{Tag: "tag", Payload: &command.List{Mailbox: "", ListMailbox: "INBOX"}}, cmd)
	require.Zero(t, *continuations)
}

func TestParser_ListMailboxAsSynchronizingLiteral(t *testing.T) {
	p, continuations := newLiteralTestParser("tag LIST \"\" {5}\r\nINBOX\r\n")

	cmd, err := p.Parse()
	require.NoError(t, err)
	require.Equal(t, command.Command{Tag: "tag", Payload: &command.List{Mailbox: "", ListMailbox: "INBOX"}}, cmd)
	require.Equal(t, 1, *continuations)
}

func TestParser_InvalidLiteralMarkers(t *testing.T) {
	for name, input := range map[string]string{
		"plus before the number": "tag LOGIN user {+4}\r\npass\r\n",
		"two plus signs":         "tag LOGIN user {4++}\r\npass\r\n",
		"minus":                  "tag LOGIN user {4-}\r\npass\r\n",
		"zero size":              "tag LOGIN user {0+}\r\n\r\n",
	} {
		t.Run(name, func(t *testing.T) {
			p, _ := newLiteralTestParser(input)

			_, err := p.Parse()
			require.Error(t, err)
		})
	}
}

func parseAfterInvalidCommand(t *testing.T, input string) command.Command {
	t.Helper()

	p, continuations := newLiteralTestParser(input)

	_, err := p.Parse()
	require.Error(t, err)
	require.NoError(t, p.ConsumeInvalidInput())

	cmd, err := p.Parse()
	require.NoError(t, err)
	require.Zero(t, *continuations)

	return cmd
}

func TestParser_ConsumeInvalidInputSkipsNonSynchronizingLiteral(t *testing.T) {
	cmd := parseAfterInvalidCommand(t, "tag BOGUS {5+}\r\nhello\r\ntag2 NOOP\r\n")

	require.Equal(t, command.Command{Tag: "tag2", Payload: &command.Noop{}}, cmd)
}

func TestParser_ConsumeInvalidInputSkipsChainedLiterals(t *testing.T) {
	cmd := parseAfterInvalidCommand(t, "tag BOGUS {5+}\r\nhello {3+}\r\nabc tail\r\ntag2 NOOP\r\n")

	require.Equal(t, command.Command{Tag: "tag2", Payload: &command.Noop{}}, cmd)
}

func TestParser_ConsumeInvalidInputDoesNotRunCommandsInsideLiteral(t *testing.T) {
	cmd := parseAfterInvalidCommand(t, "tag BOGUS {19+}\r\nevil CREATE owned\r\n\r\ntag2 NOOP\r\n")

	require.Equal(t, command.Command{Tag: "tag2", Payload: &command.Noop{}}, cmd)
}

func TestParser_ConsumeInvalidInputSkipsLiteralWhenErrorIsAtTheMarker(t *testing.T) {
	cmd := parseAfterInvalidCommand(t, "tag FETCH 1 {5+}\r\nhello\r\ntag2 NOOP\r\n")

	require.Equal(t, command.Command{Tag: "tag2", Payload: &command.Noop{}}, cmd)
}

func TestParser_ConsumeInvalidInputLeavesSynchronizingLiteralAlone(t *testing.T) {
	cmd := parseAfterInvalidCommand(t, "tag BOGUS {5}\r\ntag2 NOOP\r\n")

	require.Equal(t, command.Command{Tag: "tag2", Payload: &command.Noop{}}, cmd)
}

func TestParser_ConsumeInvalidInputIgnoresMarkerInsideParsedLiteral(t *testing.T) {
	cmd := parseAfterInvalidCommand(t, "tag LOGIN {4+}\r\n{9+}\r\ntag2 NOOP\r\n")

	require.Equal(t, command.Command{Tag: "tag2", Payload: &command.Noop{}}, cmd)
}

func TestParser_ConsumeInvalidInputRejectsOversizedLiteral(t *testing.T) {
	for _, size := range []string{"99999999", "99999999999999999999999"} {
		p, _ := newLiteralTestParser("tag BOGUS {" + size + "+}\r\n")

		_, err := p.Parse()
		require.Error(t, err)
		require.Error(t, p.ConsumeInvalidInput())
	}
}

func TestParser_ConsumeInvalidInputOnlySkipsAfterAMarkerThatEndsTheLine(t *testing.T) {
	for _, input := range []string{
		"tag BOGUS {5+} junk\r\ntag2 NOOP\r\n",
		"tag BOGUS {+}\r\ntag2 NOOP\r\n",
		"tag BOGUS 5+}\r\ntag2 NOOP\r\n",
		"tag BOGUS junk\r\ntag2 NOOP\r\n",
	} {
		cmd := parseAfterInvalidCommand(t, input)

		require.Equal(t, command.Command{Tag: "tag2", Payload: &command.Noop{}}, cmd, input)
	}
}

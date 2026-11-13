package tests

import (
	"strings"
	"testing"

	"github.com/ProtonMail/gluon/imap/command"

	"github.com/ProtonMail/gluon/rfcparser"
	"github.com/stretchr/testify/require"
)

func TestParser_TrailingArgumentsKeepTheTagAndDropThePayload(t *testing.T) {
	for _, line := range []string{
		`A001 NOOP junk`,
		`A001 STARTTLS junk`,
		`A001 CAPABILITY junk`,
	} {
		p := newCommandParser([]string{line})

		cmd, err := p.Parse()
		require.Error(t, err)
		require.Equal(t, command.Command{Tag: "A001"}, cmd)
		require.Equal(t, "A001", p.LastParsedTag())
	}
}

func TestParser_MissingLineFeedKeepsTheTag(t *testing.T) {
	p := command.NewParser(rfcparser.NewScanner(strings.NewReader("A001 NOOP\rA002 NOOP\r\n")))

	cmd, err := p.Parse()
	require.Error(t, err)
	require.Equal(t, command.Command{Tag: "A001"}, cmd)
}

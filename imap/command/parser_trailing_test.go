package command

import (
	"bytes"
	"testing"

	"github.com/ProtonMail/gluon/rfcparser"
	"github.com/stretchr/testify/require"
)

func TestParser_TrailingArgumentsKeepTheTagAndDropThePayload(t *testing.T) {
	for _, line := range []string{
		`A001 NOOP junk`,
		`A001 STARTTLS junk`,
		`A001 CAPABILITY junk`,
	} {
		p := NewParser(rfcparser.NewScanner(bytes.NewReader(toIMAPLine(line))))

		cmd, err := p.Parse()
		require.Error(t, err)
		require.Equal(t, Command{Tag: "A001"}, cmd)
		require.Equal(t, "A001", p.LastParsedTag())
	}
}

func TestParser_MissingLineFeedKeepsTheTag(t *testing.T) {
	p := NewParser(rfcparser.NewScanner(bytes.NewReader([]byte("A001 NOOP\rA002 NOOP\r\n"))))

	cmd, err := p.Parse()
	require.Error(t, err)
	require.Equal(t, Command{Tag: "A001"}, cmd)
}

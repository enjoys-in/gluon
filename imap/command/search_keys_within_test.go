package command

import (
	"bytes"
	"testing"

	"github.com/ProtonMail/gluon/rfcparser"
	"github.com/stretchr/testify/require"
)

func TestParser_SearchCommandOlderAndYounger(t *testing.T) {
	expected := Command{Tag: "tag", Payload: &Search{
		Keys: []SearchKey{
			&SearchKeyOlder{Value: 3600},
			&SearchKeyNot{Key: &SearchKeyYounger{Value: 60}},
		},
	}}

	p := NewParser(rfcparser.NewScanner(bytes.NewReader(toIMAPLine(`tag SEARCH older 3600 NOT YOUNGER 60`))))

	cmd, err := p.Parse()
	require.NoError(t, err)
	require.Equal(t, expected, cmd)
	require.Equal(t, "OLDER 3600", SearchKeyOlder{Value: 3600}.String())
	require.Equal(t, "YOUNGER 60", SearchKeyYounger{Value: 60}.String())
}

func TestParser_SearchCommandOlderAndYoungerFailures(t *testing.T) {
	for _, line := range []string{
		`tag SEARCH OLDER`,
		`tag SEARCH OLDER 0`,
		`tag SEARCH YOUNGER 0`,
		`tag SEARCH YOUNGER soon`,
		`tag SEARCH OLDER -5`,
		`tag SEARCH YOUNGER 1-Jan-2020`,
	} {
		p := NewParser(rfcparser.NewScanner(bytes.NewReader(toIMAPLine(line))))

		_, err := p.Parse()
		require.Error(t, err, line)
	}
}

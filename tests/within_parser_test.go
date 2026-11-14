package tests

import (
	"testing"

	"github.com/ProtonMail/gluon/imap/command"

	"github.com/stretchr/testify/require"
)

func TestParser_SearchCommandOlderAndYounger(t *testing.T) {
	expected := command.Command{Tag: "tag", Payload: &command.Search{
		Keys: []command.SearchKey{
			&command.SearchKeyOlder{Value: 3600},
			&command.SearchKeyNot{Key: &command.SearchKeyYounger{Value: 60}},
		},
	}}

	p := newCommandParser([]string{`tag SEARCH older 3600 NOT YOUNGER 60`})

	cmd, err := p.Parse()
	require.NoError(t, err)
	require.Equal(t, expected, cmd)
	require.Equal(t, "OLDER 3600", command.SearchKeyOlder{Value: 3600}.String())
	require.Equal(t, "YOUNGER 60", command.SearchKeyYounger{Value: 60}.String())
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
		p := newCommandParser([]string{line})

		_, err := p.Parse()
		require.Error(t, err, line)
	}
}

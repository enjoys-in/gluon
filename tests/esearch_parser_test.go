package tests

import (
	"testing"

	"github.com/ProtonMail/gluon/imap/command"

	"github.com/stretchr/testify/require"
)

func parseSearch(t *testing.T, line string) *command.Search {
	t.Helper()

	cmd, err := parseCommand(line)
	require.NoError(t, err)

	if uid, ok := cmd.Payload.(*command.UID); ok {
		search, ok := uid.Command.(*command.Search)
		require.True(t, ok)

		return search
	}

	search, ok := cmd.Payload.(*command.Search)
	require.True(t, ok)

	return search
}

func TestParser_SearchWithoutReturn(t *testing.T) {
	search := parseSearch(t, `tag SEARCH ALL`)

	require.Empty(t, search.Return)
}

func TestParser_SearchReturnOptions(t *testing.T) {
	search := parseSearch(t, `tag SEARCH RETURN (MIN MAX COUNT) UNSEEN`)

	require.Equal(t, []command.SearchReturnOption{command.SearchReturnMin, command.SearchReturnMax, command.SearchReturnCount}, search.Return)
	require.Equal(t, []command.SearchKey{&command.SearchKeyUnseen{}}, search.Keys)
}

func TestParser_SearchReturnEmptyMeansAll(t *testing.T) {
	search := parseSearch(t, `tag SEARCH RETURN () ALL`)

	require.Equal(t, []command.SearchReturnOption{command.SearchReturnAll}, search.Return)
	require.Equal(t, []command.SearchKey{&command.SearchKeyAll{}}, search.Keys)
}

func TestParser_SearchReturnIsCaseInsensitive(t *testing.T) {
	search := parseSearch(t, `tag search return (all Count) all`)

	require.Equal(t, []command.SearchReturnOption{command.SearchReturnAll, command.SearchReturnCount}, search.Return)
}

func TestParser_SearchReturnDropsDuplicates(t *testing.T) {
	search := parseSearch(t, `tag SEARCH RETURN (MIN MIN MAX MIN) ALL`)

	require.Equal(t, []command.SearchReturnOption{command.SearchReturnMin, command.SearchReturnMax}, search.Return)
}

func TestParser_SearchReturnWithCharset(t *testing.T) {
	search := parseSearch(t, `tag SEARCH RETURN (ALL) CHARSET UTF-8 UNSEEN`)

	require.Equal(t, "UTF-8", search.Charset)
	require.Equal(t, []command.SearchKey{&command.SearchKeyUnseen{}}, search.Keys)
}

func TestParser_SearchReturnWithSeveralKeys(t *testing.T) {
	search := parseSearch(t, `tag SEARCH RETURN (COUNT) UNSEEN RECENT`)

	require.Equal(t, []command.SearchKey{&command.SearchKeyUnseen{}, &command.SearchKeyRecent{}}, search.Keys)
}

func TestParser_UIDSearchReturn(t *testing.T) {
	search := parseSearch(t, `tag UID SEARCH RETURN (MAX) ALL`)

	require.Equal(t, []command.SearchReturnOption{command.SearchReturnMax}, search.Return)
}

func TestParser_SearchRecentIsNotReturn(t *testing.T) {
	search := parseSearch(t, `tag SEARCH RECENT`)

	require.Equal(t, []command.SearchKey{&command.SearchKeyRecent{}}, search.Keys)
}

func TestParser_SearchReturnInvalid(t *testing.T) {
	for name, line := range map[string]string{
		"unknown option":               `tag SEARCH RETURN (BOGUS) ALL`,
		"options without parentheses":  `tag SEARCH RETURN MIN ALL`,
		"unterminated options":         `tag SEARCH RETURN (MIN ALL`,
		"no search keys":               `tag SEARCH RETURN (MIN)`,
		"return twice":                 `tag SEARCH RETURN (MIN) RETURN (MAX) ALL`,
		"return after the search keys": `tag SEARCH ALL RETURN (MIN)`,
		"return without anything":      `tag SEARCH RETURN`,
	} {
		t.Run(name, func(t *testing.T) {
			_, err := parseCommand(line)
			require.Error(t, err)
		})
	}
}

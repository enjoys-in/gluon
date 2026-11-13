package command

import (
	"testing"

	"github.com/stretchr/testify/require"
)

var searchResultSeqSet = []SeqRange{{Begin: SeqNumValueSearchResult, End: SeqNumValueSearchResult}}

func TestParser_SearchResultInSequenceSets(t *testing.T) {
	for line, seqSet := range map[string]func(Payload) []SeqRange{
		`tag FETCH $ (FLAGS)`:        func(p Payload) []SeqRange { return p.(*Fetch).SeqSet },
		`tag STORE $ +FLAGS (\Seen)`: func(p Payload) []SeqRange { return p.(*Store).SeqSet },
		`tag COPY $ Archive`:         func(p Payload) []SeqRange { return p.(*Copy).SeqSet },
		`tag MOVE $ Archive`:         func(p Payload) []SeqRange { return p.(*Move).SeqSet },
		`tag UID FETCH $ (FLAGS)`:    func(p Payload) []SeqRange { return p.(*UID).Command.(*Fetch).SeqSet },
		`tag UID EXPUNGE $`:          func(p Payload) []SeqRange { return p.(*UIDExpunge).SeqSet },
		`tag SEARCH $`:               func(p Payload) []SeqRange { return p.(*Search).Keys[0].(*SearchKeySeqSet).SeqSet },
		`tag SEARCH UNSEEN $`:        func(p Payload) []SeqRange { return p.(*Search).Keys[1].(*SearchKeySeqSet).SeqSet },
		`tag SEARCH UID $`:           func(p Payload) []SeqRange { return p.(*Search).Keys[0].(*SearchKeyUID).SeqSet },
		`tag SEARCH NOT $`:           func(p Payload) []SeqRange { return p.(*Search).Keys[0].(*SearchKeyNot).Key.(*SearchKeySeqSet).SeqSet },
		`tag SORT (DATE) UTF-8 $`:    func(p Payload) []SeqRange { return p.(*Sort).Keys[0].(*SearchKeySeqSet).SeqSet },
		`tag SEARCH RETURN (SAVE) $`: func(p Payload) []SeqRange { return p.(*Search).Keys[0].(*SearchKeySeqSet).SeqSet },
		`tag SEARCH RETURN (COUNT) ($)`: func(p Payload) []SeqRange {
			return p.(*Search).Keys[0].(*SearchKeyList).Keys[0].(*SearchKeySeqSet).SeqSet
		},
	} {
		cmd, err := testParseCommand(line)
		require.NoError(t, err, line)
		require.Equal(t, searchResultSeqSet, seqSet(cmd.Payload), line)
	}

	require.True(t, IsSearchResultSeqSet(searchResultSeqSet))
	require.False(t, IsSearchResultSeqSet([]SeqRange{{Begin: 1, End: SeqNumValueAsterisk}}))
	require.False(t, IsSearchResultSeqSet(nil))
	require.Equal(t, "$", SeqNumValueSearchResult.String())
}

func TestParser_SearchResultCannotBeCombined(t *testing.T) {
	for _, line := range []string{
		`tag FETCH $,1 (FLAGS)`,
		`tag FETCH 1,$ (FLAGS)`,
		`tag FETCH $:* (FLAGS)`,
		`tag FETCH 1:$ (FLAGS)`,
		`tag STORE $$ +FLAGS (\Seen)`,
		`tag SEARCH UID 1,$`,
	} {
		_, err := testParseCommand(line)
		require.Error(t, err, line)
	}
}

func TestParser_SearchReturnSave(t *testing.T) {
	search := parseSearch(t, `tag SEARCH RETURN (SAVE) UNSEEN`)
	require.Equal(t, []SearchReturnOption{SearchReturnSave}, search.Return)

	search = parseSearch(t, `tag SEARCH RETURN (save MIN COUNT) UNSEEN`)
	require.Equal(t, []SearchReturnOption{SearchReturnSave, SearchReturnMin, SearchReturnCount}, search.Return)
}

func TestParser_SortCannotSave(t *testing.T) {
	_, err := testParseCommand(`tag SORT RETURN (SAVE) (DATE) UTF-8 ALL`)
	require.Error(t, err)

	_, err = testParseCommand(`tag SORT RETURN (ALL SAVE) (DATE) UTF-8 ALL`)
	require.Error(t, err)
}

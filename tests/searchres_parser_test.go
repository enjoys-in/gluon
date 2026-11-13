package tests

import (
	"testing"

	"github.com/ProtonMail/gluon/imap/command"

	"github.com/stretchr/testify/require"
)

var searchResultSeqSet = []command.SeqRange{{Begin: command.SeqNumValueSearchResult, End: command.SeqNumValueSearchResult}}

func TestParser_SearchResultInSequenceSets(t *testing.T) {
	for line, seqSet := range map[string]func(command.Payload) []command.SeqRange{
		`tag FETCH $ (FLAGS)`:        func(p command.Payload) []command.SeqRange { return p.(*command.Fetch).SeqSet },
		`tag STORE $ +FLAGS (\Seen)`: func(p command.Payload) []command.SeqRange { return p.(*command.Store).SeqSet },
		`tag COPY $ Archive`:         func(p command.Payload) []command.SeqRange { return p.(*command.Copy).SeqSet },
		`tag MOVE $ Archive`:         func(p command.Payload) []command.SeqRange { return p.(*command.Move).SeqSet },
		`tag UID FETCH $ (FLAGS)`:    func(p command.Payload) []command.SeqRange { return p.(*command.UID).Command.(*command.Fetch).SeqSet },
		`tag UID EXPUNGE $`:          func(p command.Payload) []command.SeqRange { return p.(*command.UIDExpunge).SeqSet },
		`tag SEARCH $`: func(p command.Payload) []command.SeqRange {
			return p.(*command.Search).Keys[0].(*command.SearchKeySeqSet).SeqSet
		},
		`tag SEARCH UNSEEN $`: func(p command.Payload) []command.SeqRange {
			return p.(*command.Search).Keys[1].(*command.SearchKeySeqSet).SeqSet
		},
		`tag SEARCH UID $`: func(p command.Payload) []command.SeqRange {
			return p.(*command.Search).Keys[0].(*command.SearchKeyUID).SeqSet
		},
		`tag SEARCH NOT $`: func(p command.Payload) []command.SeqRange {
			return p.(*command.Search).Keys[0].(*command.SearchKeyNot).Key.(*command.SearchKeySeqSet).SeqSet
		},
		`tag SORT (DATE) UTF-8 $`: func(p command.Payload) []command.SeqRange {
			return p.(*command.Sort).Keys[0].(*command.SearchKeySeqSet).SeqSet
		},
		`tag SEARCH RETURN (SAVE) $`: func(p command.Payload) []command.SeqRange {
			return p.(*command.Search).Keys[0].(*command.SearchKeySeqSet).SeqSet
		},
		`tag SEARCH RETURN (COUNT) ($)`: func(p command.Payload) []command.SeqRange {
			return p.(*command.Search).Keys[0].(*command.SearchKeyList).Keys[0].(*command.SearchKeySeqSet).SeqSet
		},
	} {
		cmd, err := parseCommand(line)
		require.NoError(t, err, line)
		require.Equal(t, searchResultSeqSet, seqSet(cmd.Payload), line)
	}

	require.True(t, command.IsSearchResultSeqSet(searchResultSeqSet))
	require.False(t, command.IsSearchResultSeqSet([]command.SeqRange{{Begin: 1, End: command.SeqNumValueAsterisk}}))
	require.False(t, command.IsSearchResultSeqSet(nil))
	require.Equal(t, "$", command.SeqNumValueSearchResult.String())
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
		_, err := parseCommand(line)
		require.Error(t, err, line)
	}
}

func TestParser_SearchReturnSave(t *testing.T) {
	search := parseSearch(t, `tag SEARCH RETURN (SAVE) UNSEEN`)
	require.Equal(t, []command.SearchReturnOption{command.SearchReturnSave}, search.Return)

	search = parseSearch(t, `tag SEARCH RETURN (save MIN COUNT) UNSEEN`)
	require.Equal(t, []command.SearchReturnOption{command.SearchReturnSave, command.SearchReturnMin, command.SearchReturnCount}, search.Return)
}

func TestParser_SortCannotSave(t *testing.T) {
	_, err := parseCommand(`tag SORT RETURN (SAVE) (DATE) UTF-8 ALL`)
	require.Error(t, err)

	_, err = parseCommand(`tag SORT RETURN (ALL SAVE) (DATE) UTF-8 ALL`)
	require.Error(t, err)
}

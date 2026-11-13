package state

import (
	"context"
	"testing"

	"github.com/ProtonMail/gluon/imap"
	"github.com/ProtonMail/gluon/imap/command"
	"github.com/ProtonMail/gluon/internal/contexts"
	"github.com/stretchr/testify/require"
)

func newSearchResultSnapshot(t *testing.T, uids ...imap.UID) *snapshot {
	snap := &snapshot{messages: newMsgList(len(uids))}

	for _, uid := range uids {
		require.NoError(t, snap.messages.insert(messageIDPair(imap.NewInternalMessageID(), ""), uid, imap.NewFlagSet()))
	}

	return snap
}

var searchResultSeqSet = []command.SeqRange{{Begin: command.SeqNumValueSearchResult, End: command.SeqNumValueSearchResult}}

func TestSnapshotSearchResult(t *testing.T) {
	snap := newSearchResultSnapshot(t, 10, 20, 30, 40, 50, 60)

	// 35 is not in the mailbox, for example because it was expunged after the search.
	snap.setSearchResult([]imap.UID{20, 30, 35, 40, 60})

	seqs, err := snap.resolveSeqInterval(searchResultSeqSet)
	require.NoError(t, err)
	require.Equal(t, []SeqInterval{{begin: 2, end: 4}, {begin: 6, end: 6}}, seqs)

	uids, err := snap.resolveUIDInterval(searchResultSeqSet)
	require.NoError(t, err)
	require.Equal(t, []UIDInterval{{begin: 20, end: 40}, {begin: 60, end: 60}}, uids)

	for _, ctx := range []context.Context{context.Background(), contexts.AsUID(context.Background())} {
		messages, err := snap.getMessagesInRange(ctx, searchResultSeqSet)
		require.NoError(t, err)
		require.Len(t, messages, 4)
		require.Equal(t, imap.SeqID(2), messages[0].Seq)
		require.Equal(t, imap.UID(60), messages[3].UID)
	}
}

func TestSnapshotSearchResultEmpty(t *testing.T) {
	for _, snap := range []*snapshot{newSearchResultSnapshot(t), newSearchResultSnapshot(t, 10, 20)} {
		seqs, err := snap.resolveSeqInterval(searchResultSeqSet)
		require.NoError(t, err)
		require.Empty(t, seqs)

		uids, err := snap.resolveUIDInterval(searchResultSeqSet)
		require.NoError(t, err)
		require.Empty(t, uids)

		messages, err := snap.getMessagesInRange(context.Background(), searchResultSeqSet)
		require.NoError(t, err)
		require.Empty(t, messages)
	}
}

func TestSnapshotOtherSequenceSetsAreUnchanged(t *testing.T) {
	snap := newSearchResultSnapshot(t, 10, 20, 30)
	snap.setSearchResult([]imap.UID{30})

	messages, err := snap.getMessagesInRange(context.Background(), []command.SeqRange{{Begin: 1, End: 2}})
	require.NoError(t, err)
	require.Len(t, messages, 2)
}

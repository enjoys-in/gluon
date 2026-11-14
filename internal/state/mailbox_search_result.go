package state

import (
	"context"

	"github.com/ProtonMail/gluon/imap"
	"github.com/ProtonMail/gluon/internal/contexts"
)

// SaveSearchResult remembers the messages a search found, so that later commands can refer to them with "$".
// The numbers are UIDs for a UID command and sequence numbers otherwise, as Search returns them.
func (m *Mailbox) SaveSearchResult(ctx context.Context, seq []uint32) {
	uids := make([]imap.UID, 0, len(seq))

	for _, number := range seq {
		if contexts.IsUID(ctx) {
			uids = append(uids, imap.UID(number))
		} else if msg, ok := m.snap.messages.getWithSeqID(imap.SeqID(number)); ok {
			uids = append(uids, msg.UID)
		}
	}

	m.snap.setSearchResult(uids)
}

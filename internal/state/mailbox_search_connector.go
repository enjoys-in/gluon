package state

import (
	"context"

	"github.com/ProtonMail/gluon/imap"
	"github.com/ProtonMail/gluon/imap/command"
	"github.com/ProtonMail/gluon/internal/contexts"
	"golang.org/x/text/encoding"
)

type searchKeyOwner int

const (
	searchKeyOwnerGluon searchKeyOwner = iota
	searchKeyOwnerConnector
	searchKeyOwnerMixed
)

// SearchWithConnector lets the connector answer the keys about message content. Gluon applies the keys that depend
// on its own sequence numbers, UIDs and flags. It returns false when Gluon has to run the whole search itself.
func (m *Mailbox) SearchWithConnector(ctx context.Context, keys []command.SearchKey, decoder *encoding.Decoder) ([]uint32, bool, error) {
	if m.state.user.GetRecoveryMailboxID().InternalID == m.snap.mboxID.InternalID {
		return nil, false, nil
	}

	var content, local []command.SearchKey

	for _, key := range keys {
		switch getSearchKeyOwner(key) {
		case searchKeyOwnerConnector:
			content = append(content, key)

		case searchKeyOwnerGluon:
			local = append(local, key)

		case searchKeyOwnerMixed:
			return nil, false, nil
		}
	}

	if len(content) == 0 {
		return nil, false, nil
	}

	messageIDs, ok, err := m.state.user.GetRemote().SearchMessages(ctx, m.snap.mboxID.RemoteID, content)
	if err != nil || !ok {
		return nil, false, err
	}

	found := make(map[imap.MessageID]struct{}, len(messageIDs))
	for _, messageID := range messageIDs {
		found[messageID] = struct{}{}
	}

	op, err := buildSearchOpListWithKeys(m, local, decoder)
	if err != nil {
		return nil, false, err
	}

	var result []uint32

	for i := range m.snap.len() {
		msg, ok := m.snap.messages.getWithSeqID(imap.SeqID(uint32(i + 1)))
		if !ok {
			continue
		}

		if _, ok := found[msg.ID.RemoteID]; !ok {
			continue
		}

		matches, err := applySearch(ctx, m, msg, op)
		if err != nil {
			return nil, false, err
		}

		if !matches {
			continue
		}

		if contexts.IsUID(ctx) {
			result = append(result, uint32(msg.UID))
		} else {
			result = append(result, uint32(msg.Seq))
		}
	}

	return result, true, nil
}

func getSearchKeyOwner(key command.SearchKey) searchKeyOwner {
	switch key := key.(type) {
	case
		*command.SearchKeyBCC,
		*command.SearchKeyBefore,
		*command.SearchKeyBody,
		*command.SearchKeyCC,
		*command.SearchKeyFrom,
		*command.SearchKeyHeader,
		*command.SearchKeyOn,
		*command.SearchKeySentBefore,
		*command.SearchKeySentOn,
		*command.SearchKeySentSince,
		*command.SearchKeySince,
		*command.SearchKeySubject,
		*command.SearchKeyText,
		*command.SearchKeyTo:
		return searchKeyOwnerConnector

	case *command.SearchKeyNot:
		return getSearchKeyOwner(key.Key)

	case *command.SearchKeyOr:
		return getSearchKeysOwner(key.Key1, key.Key2)

	case *command.SearchKeyList:
		return getSearchKeysOwner(key.Keys...)

	default:
		return searchKeyOwnerGluon
	}
}

func getSearchKeysOwner(keys ...command.SearchKey) searchKeyOwner {
	owner := searchKeyOwnerGluon

	for i, key := range keys {
		keyOwner := getSearchKeyOwner(key)

		if i == 0 {
			owner = keyOwner
		} else if keyOwner != owner {
			return searchKeyOwnerMixed
		}
	}

	return owner
}

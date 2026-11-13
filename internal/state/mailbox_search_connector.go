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
	found, local, ok, err := m.searchConnector(ctx, keys, decoder)
	if err != nil || !ok {
		return nil, false, err
	}

	if len(found) == 0 {
		return nil, true, nil
	}

	var op *buildSearchOpResult

	if len(local) > 0 {
		if op, err = buildSearchOpListWithKeys(m, local, decoder); err != nil {
			return nil, false, err
		}
	}

	var (
		isUID  = contexts.IsUID(ctx)
		result = make([]uint32, 0, min(len(found), m.snap.len()))
	)

	for i := range m.snap.len() {
		msg, ok := m.snap.messages.getWithSeqID(imap.SeqID(uint32(i + 1)))
		if !ok {
			continue
		}

		if _, ok := found[msg.ID.RemoteID]; !ok {
			continue
		}

		if op != nil {
			matches, err := applySearch(ctx, m, msg, op)
			if err != nil {
				return nil, false, err
			}

			if !matches {
				continue
			}
		}

		if isUID {
			result = append(result, uint32(msg.UID))
		} else {
			result = append(result, uint32(msg.Seq))
		}
	}

	return result, true, nil
}

// searchConnector asks the connector for the messages that match the keys about message content. It returns the
// connector's answer and the keys left for Gluon, or false when Gluon has to run the whole search itself.
func (m *Mailbox) searchConnector(ctx context.Context, keys []command.SearchKey, decoder *encoding.Decoder) (map[imap.MessageID]struct{}, []command.SearchKey, bool, error) {
	if m.state.user.GetRecoveryMailboxID().InternalID == m.snap.mboxID.InternalID {
		return nil, nil, false, nil
	}

	var content, local []command.SearchKey

	for _, key := range keys {
		switch getSearchKeyOwner(key) {
		case searchKeyOwnerConnector:
			content = append(content, key)

		case searchKeyOwnerGluon:
			local = append(local, key)

		case searchKeyOwnerMixed:
			return nil, nil, false, nil
		}
	}

	if len(content) == 0 {
		return nil, nil, false, nil
	}

	content, err := decodeSearchKeys(content, decoder)
	if err != nil {
		return nil, nil, false, err
	}

	messageIDs, ok, err := m.state.user.GetRemote().SearchMessages(ctx, m.snap.mboxID.RemoteID, content)
	if err != nil || !ok {
		return nil, nil, false, err
	}

	found := make(map[imap.MessageID]struct{}, len(messageIDs))
	for _, messageID := range messageIDs {
		found[messageID] = struct{}{}
	}

	return found, local, true, nil
}

// buildSearchOpWithConnector builds the op for the keys like buildSearchOpListWithKeys. When the connector answers
// the keys about message content, the op checks its answer instead of reading those messages.
func buildSearchOpWithConnector(ctx context.Context, m *Mailbox, keys []command.SearchKey, decoder *encoding.Decoder) (*buildSearchOpResult, error) {
	found, local, ok, err := m.searchConnector(ctx, keys, decoder)
	if err != nil {
		return nil, err
	}

	if !ok {
		return buildSearchOpListWithKeys(m, keys, decoder)
	}

	result, err := buildSearchOpListWithKeys(m, local, decoder)
	if err != nil {
		return nil, err
	}

	localOp := result.op

	result.op = func(s *searchData) (bool, error) {
		if _, ok := found[s.message.ID.RemoteID]; !ok {
			return false, nil
		}

		return localOp(s)
	}

	return result, nil
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

// decodeSearchKeys converts the search strings of the keys to UTF-8, the only charset the connector is given.
func decodeSearchKeys(keys []command.SearchKey, decoder *encoding.Decoder) ([]command.SearchKey, error) {
	decoded := make([]command.SearchKey, 0, len(keys))

	for _, key := range keys {
		key, err := decodeSearchKey(key, decoder)
		if err != nil {
			return nil, err
		}

		decoded = append(decoded, key)
	}

	return decoded, nil
}

func decodeSearchKey(key command.SearchKey, decoder *encoding.Decoder) (command.SearchKey, error) {
	switch key := key.(type) {
	case *command.SearchKeyBCC:
		value, err := decoder.String(key.Value)
		return &command.SearchKeyBCC{Value: value}, err

	case *command.SearchKeyBody:
		value, err := decoder.String(key.Value)
		return &command.SearchKeyBody{Value: value}, err

	case *command.SearchKeyCC:
		value, err := decoder.String(key.Value)
		return &command.SearchKeyCC{Value: value}, err

	case *command.SearchKeyFrom:
		value, err := decoder.String(key.Value)
		return &command.SearchKeyFrom{Value: value}, err

	case *command.SearchKeyHeader:
		value, err := decoder.String(key.Value)
		return &command.SearchKeyHeader{Field: key.Field, Value: value}, err

	case *command.SearchKeySubject:
		value, err := decoder.String(key.Value)
		return &command.SearchKeySubject{Value: value}, err

	case *command.SearchKeyText:
		value, err := decoder.String(key.Value)
		return &command.SearchKeyText{Value: value}, err

	case *command.SearchKeyTo:
		value, err := decoder.String(key.Value)
		return &command.SearchKeyTo{Value: value}, err

	case *command.SearchKeyNot:
		inner, err := decodeSearchKey(key.Key, decoder)
		return &command.SearchKeyNot{Key: inner}, err

	case *command.SearchKeyOr:
		keys, err := decodeSearchKeys([]command.SearchKey{key.Key1, key.Key2}, decoder)
		if err != nil {
			return nil, err
		}

		return &command.SearchKeyOr{Key1: keys[0], Key2: keys[1]}, nil

	case *command.SearchKeyList:
		keys, err := decodeSearchKeys(key.Keys, decoder)
		return &command.SearchKeyList{Keys: keys}, err

	default:
		return key, nil
	}
}

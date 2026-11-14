package connector

import (
	"bytes"
	"context"
	"strings"

	"github.com/ProtonMail/gluon/imap"
	"github.com/ProtonMail/gluon/imap/command"
	"github.com/ProtonMail/gluon/rfc822"
)

// SearchMessages answers header and text searches from the connector's own messages, the way a mail store with its
// own index would. Keys it cannot evaluate are left to Gluon.
func (conn *Dummy) SearchMessages(_ context.Context, mboxID imap.MailboxID, keys []command.SearchKey) ([]imap.MessageID, bool, error) {
	if !conn.searchEnabled {
		return nil, false, nil
	}

	return conn.state.searchMessages(mboxID, keys)
}

func (conn *Dummy) SetSearchEnabled(value bool) {
	conn.searchEnabled = value
}

func (state *dummyState) searchMessages(mboxID imap.MailboxID, keys []command.SearchKey) ([]imap.MessageID, bool, error) {
	state.lock.RLock()
	defer state.lock.RUnlock()

	if _, ok := state.mailboxes[mboxID]; !ok {
		return nil, false, ErrNoSuchMailbox
	}

	var messageIDs []imap.MessageID

	for messageID, message := range state.messages {
		if _, ok := message.mboxIDs[mboxID]; !ok {
			continue
		}

		matches, ok := message.matches(&command.SearchKeyList{Keys: keys})
		if !ok {
			return nil, false, nil
		}

		if matches {
			messageIDs = append(messageIDs, messageID)
		}
	}

	return messageIDs, true, nil
}

// matches reports whether the message matches the key, and false when the key is not one the connector can evaluate.
func (message *dummyMessage) matches(key command.SearchKey) (bool, bool) {
	switch key := key.(type) {
	case *command.SearchKeySubject:
		return message.headerContains("Subject", key.Value), true

	case *command.SearchKeyFrom:
		return message.headerContains("From", key.Value), true

	case *command.SearchKeyTo:
		return message.headerContains("To", key.Value), true

	case *command.SearchKeyCC:
		return message.headerContains("Cc", key.Value), true

	case *command.SearchKeyBCC:
		return message.headerContains("Bcc", key.Value), true

	case *command.SearchKeyHeader:
		return message.headerContains(key.Field, key.Value), true

	case *command.SearchKeyBody:
		_, body := rfc822.Split(message.literal)

		return containsFold(body, key.Value), true

	case *command.SearchKeyText:
		return containsFold(message.literal, key.Value), true

	case *command.SearchKeyNot:
		matches, ok := message.matches(key.Key)

		return !matches, ok

	case *command.SearchKeyOr:
		left, ok := message.matches(key.Key1)
		if !ok {
			return false, false
		}

		right, ok := message.matches(key.Key2)

		return left || right, ok

	case *command.SearchKeyList:
		for _, key := range key.Keys {
			if matches, ok := message.matches(key); !ok || !matches {
				return false, ok
			}
		}

		return true, true

	default:
		return false, false
	}
}

func (message *dummyMessage) headerContains(field, value string) bool {
	headerBytes, _ := rfc822.Split(message.literal)

	header, err := rfc822.NewHeader(headerBytes)
	if err != nil {
		return false
	}

	return strings.Contains(strings.ToLower(header.Get(field)), strings.ToLower(value))
}

func containsFold(data []byte, value string) bool {
	return bytes.Contains(bytes.ToLower(data), bytes.ToLower([]byte(value)))
}

package session

import (
	"errors"

	"github.com/ProtonMail/gluon/imap/command"
)

var ErrUIDRequired = errors.New("message numbers are not allowed once UIDONLY is enabled")

// UIDOnly tells whether the client has enabled UIDONLY (RFC 9586). The responses then name messages by their UIDs.
func (s *Session) UIDOnly() bool {
	return s.uidOnly.Load()
}

// usesSequenceNumbers tells whether a command names messages by their sequence numbers, which RFC 9586 does not
// allow once UIDONLY is enabled: FETCH, STORE, COPY, MOVE, SEARCH, SORT and THREAD without UID, and a sequence set
// used as a search key.
func usesSequenceNumbers(cmd command.Payload) bool {
	switch cmd := cmd.(type) {
	case *command.Fetch, *command.Store, *command.Copy, *command.Move, *command.Search, *command.Sort, *command.Thread:
		return true

	case *command.UID:
		switch cmd := cmd.Command.(type) {
		case *command.Search:
			return hasSequenceSetKey(cmd.Keys...)

		case *command.Sort:
			return hasSequenceSetKey(cmd.Keys...)

		case *command.Thread:
			return hasSequenceSetKey(cmd.Keys...)
		}
	}

	return false
}

func hasSequenceSetKey(keys ...command.SearchKey) bool {
	for _, key := range keys {
		switch key := key.(type) {
		case *command.SearchKeySeqSet:
			return true

		case *command.SearchKeyNot:
			if hasSequenceSetKey(key.Key) {
				return true
			}

		case *command.SearchKeyOr:
			if hasSequenceSetKey(key.Key1, key.Key2) {
				return true
			}

		case *command.SearchKeyList:
			if hasSequenceSetKey(key.Keys...) {
				return true
			}
		}
	}

	return false
}

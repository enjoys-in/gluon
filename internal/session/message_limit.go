package session

import (
	"context"
	"errors"
	"fmt"

	"github.com/ProtonMail/gluon/imap"
	"github.com/ProtonMail/gluon/imap/command"
	"github.com/ProtonMail/gluon/internal/response"
	"github.com/ProtonMail/gluon/internal/state"
)

var ErrTooManyMessages = errors.New("too many messages, try a smaller set")

// SetMessageLimit makes the session process at most limit messages in one command and advertise it (RFC 9738).
func (s *Session) SetMessageLimit(limit int) {
	s.messageLimit = limit

	s.addCapability(imap.Capability(fmt.Sprintf("%v=%d", imap.MessageLimit, limit)))
}

// limitSeqSet narrows the message set of a command to the messageLimit messages with the highest UIDs. The item is
// the MESSAGELIMIT response code for the tagged response, and is nil when the command was not limited.
func (s *Session) limitSeqSet(ctx context.Context, mailbox *state.Mailbox, seqSet []command.SeqRange, deletedOnly bool) ([]command.SeqRange, response.Item) {
	if s.messageLimit == 0 {
		return seqSet, nil
	}

	limited, lowest, ok := mailbox.LimitSeqSet(ctx, seqSet, s.messageLimit, deletedOnly)
	if !ok {
		return seqSet, nil
	}

	return limited, response.ItemMessageLimit(s.messageLimit, lowest)
}

// limitSearchResult keeps the messageLimit highest messages of a search result.
func (s *Session) limitSearchResult(ctx context.Context, mailbox *state.Mailbox, result []uint32) ([]uint32, response.Item) {
	if s.messageLimit == 0 {
		return result, nil
	}

	limited, lowest, ok := mailbox.LimitSearchResult(ctx, result, s.messageLimit)
	if !ok {
		return result, nil
	}

	return limited, response.ItemMessageLimit(s.messageLimit, lowest)
}

// withItem adds an item to a list of response items when there is one.
func withItem(items []response.Item, item response.Item) []response.Item {
	if item == nil {
		return items
	}

	return append(items, item)
}

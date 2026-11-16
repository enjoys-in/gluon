package state

import (
	"context"
	"slices"

	"github.com/ProtonMail/gluon/imap"
	"github.com/ProtonMail/gluon/imap/command"
	"github.com/ProtonMail/gluon/internal/contexts"
)

// LimitSeqSet narrows a message set to the limit messages with the highest UIDs, as RFC 9738 asks, and returns the
// lowest UID that is left. The last result is false when the set already fits or cannot be resolved; the command
// then runs on the set it was given. With deletedOnly only the messages marked \Deleted are counted.
func (m *Mailbox) LimitSeqSet(ctx context.Context, seqSet []command.SeqRange, limit int, deletedOnly bool) ([]command.SeqRange, imap.UID, bool) {
	messages, err := m.snap.getMessagesInRange(ctx, seqSet)
	if err != nil {
		return seqSet, 0, false
	}

	if deletedOnly {
		messages = slices.DeleteFunc(slices.Clone(messages), func(msg snapMsgWithSeq) bool { return !msg.toExpunge })
	}

	if len(messages) <= limit {
		return seqSet, 0, false
	}

	slices.SortFunc(messages, func(a, b snapMsgWithSeq) int { return int(a.UID) - int(b.UID) })

	messages = messages[len(messages)-limit:]

	number := func(msg snapMsgWithSeq) command.SeqNum { return command.SeqNum(msg.Seq) }
	if contexts.IsUID(ctx) {
		number = func(msg snapMsgWithSeq) command.SeqNum { return command.SeqNum(msg.UID) }
	}

	ranges := []command.SeqRange{{Begin: number(messages[0]), End: number(messages[0])}}

	for _, msg := range messages[1:] {
		if last := &ranges[len(ranges)-1]; number(msg) == last.End+1 {
			last.End++
		} else {
			ranges = append(ranges, command.SeqRange{Begin: number(msg), End: number(msg)})
		}
	}

	return ranges, messages[0].UID, true
}

// LimitSearchResult keeps the limit highest messages of a search result, which holds sequence numbers or, in a UID
// context, UIDs. It returns the lowest UID that is left, and false when the result already fits.
func (m *Mailbox) LimitSearchResult(ctx context.Context, result []uint32, limit int) ([]uint32, imap.UID, bool) {
	if len(result) <= limit {
		return result, 0, false
	}

	result = slices.Clone(result)

	slices.Sort(result)

	result = result[len(result)-limit:]

	if contexts.IsUID(ctx) {
		return result, imap.UID(result[0]), true
	}

	msg, ok := m.snap.messages.seq(imap.SeqID(result[0]))
	if !ok {
		return result, 0, true
	}

	return result, msg.UID, true
}

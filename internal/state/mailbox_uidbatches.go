package state

import (
	"github.com/ProtonMail/gluon/imap"
)

// UIDRange is a range of UIDs, written High:Low.
type UIDRange struct {
	High, Low imap.UID
}

// UIDBatchCount returns how many batches of the given size the mailbox is split into.
func (m *Mailbox) UIDBatchCount(batchSize int) int {
	return (m.snap.len() + batchSize - 1) / batchSize
}

// UIDBatches splits the mailbox into batches of batchSize messages, newest first, and returns batches first to last,
// counted from one. The ranges follow each other without a gap, so the oldest one always reaches down to UID 1.
func (m *Mailbox) UIDBatches(batchSize, first, last int) []UIDRange {
	messages := m.snap.messages.all()

	last = min(last, m.UIDBatchCount(batchSize))
	if first > last {
		return nil
	}

	ranges := make([]UIDRange, 0, last-first+1)

	var high imap.UID

	if len(messages) > 0 {
		high = messages[len(messages)-1].UID
	}

	for batch := 1; batch <= last; batch++ {
		low := imap.UID(1)

		if end := batch * batchSize; end < len(messages) {
			low = messages[len(messages)-end].UID
		}

		if batch >= first {
			ranges = append(ranges, UIDRange{High: high, Low: low})
		}

		high = low - 1
	}

	return ranges
}

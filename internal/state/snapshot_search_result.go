package state

import (
	"github.com/ProtonMail/gluon/imap"
)

// setSearchResult remembers the messages that later commands refer to with "$" (RFC 5182).
func (snap *snapshot) setSearchResult(uids []imap.UID) {
	snap.searchResult = uids
}

// searchResultMessages returns the remembered messages that are still in the mailbox, in mailbox order.
func (snap *snapshot) searchResultMessages() []snapMsgWithSeq {
	messages := make([]snapMsgWithSeq, 0, len(snap.searchResult))

	for _, uid := range snap.searchResult {
		if msg, ok := snap.messages.getWithUID(uid); ok {
			messages = append(messages, msg)
		}
	}

	return messages
}

// searchResultIntervals returns the remembered messages as sequence number and UID intervals. Messages that follow
// each other in the mailbox share one interval.
func (snap *snapshot) searchResultIntervals() ([]SeqInterval, []UIDInterval) {
	var (
		seqs []SeqInterval
		uids []UIDInterval
	)

	for _, msg := range snap.searchResultMessages() {
		if last := len(seqs) - 1; last >= 0 && seqs[last].end+1 == msg.Seq {
			seqs[last].end, uids[last].end = msg.Seq, msg.UID
		} else {
			seqs = append(seqs, SeqInterval{begin: msg.Seq, end: msg.Seq})
			uids = append(uids, UIDInterval{begin: msg.UID, end: msg.UID})
		}
	}

	return seqs, uids
}

package connector

import (
	"context"

	"github.com/ProtonMail/gluon/imap"
)

// AddOrUpdateMessageFlags records flags added to messages. No update is pushed: gluon already applied the change.
func (conn *Dummy) AddOrUpdateMessageFlags(_ context.Context, _ IMAPStateWrite, messageIDs []imap.MessageID, flags imap.FlagSet) error {
	conn.state.changeMessageFlags(messageIDs, flags, true)

	return nil
}

// RemoveMessageFlags records flags removed from messages.
func (conn *Dummy) RemoveMessageFlags(_ context.Context, _ IMAPStateWrite, messageIDs []imap.MessageID, flags imap.FlagSet) error {
	conn.state.changeMessageFlags(messageIDs, flags, false)

	return nil
}

// MessageFlags returns the flags the connector was told a message has, other than \Seen, \Flagged and forwarded.
func (conn *Dummy) MessageFlags(messageID imap.MessageID) imap.FlagSet {
	return conn.state.messageFlags(messageID)
}

func (state *dummyState) changeMessageFlags(messageIDs []imap.MessageID, flags imap.FlagSet, add bool) {
	state.lock.Lock()
	defer state.lock.Unlock()

	for _, messageID := range messageIDs {
		msg, ok := state.messages[messageID]
		if !ok {
			continue
		}

		if msg.flags == nil {
			msg.flags = imap.NewFlagSet()
		}

		if add {
			msg.flags = msg.flags.AddFlagSet(flags)
		} else {
			msg.flags = msg.flags.RemoveFlagSet(flags)
		}
	}
}

func (state *dummyState) messageFlags(messageID imap.MessageID) imap.FlagSet {
	state.lock.RLock()
	defer state.lock.RUnlock()

	msg, ok := state.messages[messageID]
	if !ok || msg.flags == nil {
		return imap.NewFlagSet()
	}

	return msg.flags.Clone()
}

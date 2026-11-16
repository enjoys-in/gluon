package connector

import (
	"context"

	"github.com/ProtonMail/gluon/imap"
)

func (conn *Dummy) SetMailboxSubscribed(_ context.Context, _ IMAPStateWrite, mboxID imap.MailboxID, subscribed bool) error {
	return conn.state.setMailboxSubscribed(mboxID, subscribed)
}

// MailboxSubscribed simulates a subscription that changed on the remote.
func (conn *Dummy) MailboxSubscribed(mboxID imap.MailboxID, subscribed bool) error {
	if err := conn.state.setMailboxSubscribed(mboxID, subscribed); err != nil {
		return err
	}

	conn.pushUpdate(imap.NewMailboxSubscribed(mboxID, subscribed))

	return nil
}

func (conn *Dummy) IsMailboxSubscribed(mboxID imap.MailboxID) bool {
	return conn.state.isMailboxSubscribed(mboxID)
}

func (state *dummyState) setMailboxSubscribed(mboxID imap.MailboxID, subscribed bool) error {
	state.lock.Lock()
	defer state.lock.Unlock()

	mailbox, ok := state.mailboxes[mboxID]
	if !ok {
		return ErrNoSuchMailbox
	}

	mailbox.unsubscribed = !subscribed

	return nil
}

func (state *dummyState) isMailboxSubscribed(mboxID imap.MailboxID) bool {
	state.lock.RLock()
	defer state.lock.RUnlock()

	mailbox, ok := state.mailboxes[mboxID]

	return ok && !mailbox.unsubscribed
}

package connector

import (
	"context"

	"github.com/ProtonMail/gluon/imap"
)

func (conn *Dummy) CreateSpecialUseMailbox(_ context.Context, _ IMAPStateWrite, name []string, attributes imap.FlagSet) (imap.Mailbox, error) {
	exclusive, err := conn.validateName(name)
	if err != nil {
		return imap.Mailbox{}, err
	}

	// Like a real server with one mailbox per role, the dummy refuses a special use that is already taken.
	if conn.state.hasSpecialUse(attributes) {
		return imap.Mailbox{}, ErrSpecialUseNotSupported
	}

	mbox := conn.state.createSpecialUseMailbox(name, exclusive, attributes)

	conn.pushUpdate(imap.NewMailboxCreated(mbox))

	return mbox, nil
}

func (state *dummyState) createSpecialUseMailbox(name []string, exclusive bool, attributes imap.FlagSet) imap.Mailbox {
	mbox := state.createMailbox(name, exclusive)

	state.lock.Lock()
	defer state.lock.Unlock()

	state.mailboxes[mbox.ID].specialUse = attributes

	return state.toMailbox(mbox.ID)
}

func (state *dummyState) hasSpecialUse(attributes imap.FlagSet) bool {
	state.lock.RLock()
	defer state.lock.RUnlock()

	for _, mailbox := range state.mailboxes {
		for _, attribute := range attributes.ToSlice() {
			if mailbox.specialUse.Contains(attribute) {
				return true
			}
		}
	}

	return false
}

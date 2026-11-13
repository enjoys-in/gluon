package state

import (
	"context"
	"errors"
	"slices"

	"github.com/ProtonMail/gluon/db"
	"github.com/ProtonMail/gluon/imap"
	"github.com/ProtonMail/gluon/imap/command"
)

type MailboxStatus struct {
	Messages    int
	Recent      int
	Unseen      int
	UIDNext     imap.UID
	UIDValidity imap.UID
	Size        int
}

// MailboxStatus counts with queries instead of loading the mailbox's messages. It returns false for the selected
// mailbox, whose status has to match what the session has already been told.
func (state *State) MailboxStatus(ctx context.Context, name string, attributes []command.StatusAttribute) (MailboxStatus, bool, error) {
	type result struct {
		status MailboxStatus
		ok     bool
	}

	res, err := stateDBReadResult(ctx, state, func(ctx context.Context, client db.ReadOnly) (result, error) {
		mbox, err := client.GetMailboxByName(ctx, name)
		if err != nil {
			return result{}, err
		}

		if state.snap != nil && state.snap.mboxID.InternalID == mbox.ID {
			return result{}, nil
		}

		status := MailboxStatus{UIDValidity: mbox.UIDValidity}

		for _, attribute := range attributes {
			switch attribute {
			case command.StatusAttributeMessages:
				status.Messages, err = client.GetMailboxMessageCount(ctx, mbox.ID)

			case command.StatusAttributeRecent:
				status.Recent, err = client.GetMailboxRecentCount(ctx, mbox.ID)

			case command.StatusAttributeUnseen:
				status.Unseen, err = client.GetMailboxUnseenCount(ctx, mbox.ID)

			case command.StatusAttributeUIDNext:
				status.UIDNext, err = client.GetMailboxUID(ctx, mbox.ID)

			case command.StatusAttributeSize:
				status.Size, err = client.GetMailboxSize(ctx, mbox.ID)

			case command.StatusAttributeUIDValidity:
			}

			if err != nil {
				return result{}, err
			}
		}

		return result{status: status, ok: true}, nil
	})
	if errors.Is(err, db.ErrNotFound) {
		return MailboxStatus{}, false, ErrNoSuchMailbox
	}

	return res.status, res.ok, err
}

// Status reports the mailbox as the session sees it.
func (m *Mailbox) Status(ctx context.Context, attributes []command.StatusAttribute) (MailboxStatus, error) {
	uidNext, err := m.UIDNext(ctx)
	if err != nil {
		return MailboxStatus{}, err
	}

	status := MailboxStatus{
		Messages:    m.Count(),
		Recent:      m.GetMessagesWithFlagCount(imap.FlagRecent),
		Unseen:      m.GetMessagesWithoutFlagCount(imap.FlagSeen),
		UIDNext:     uidNext,
		UIDValidity: m.UIDValidity(),
	}

	if slices.Contains(attributes, command.StatusAttributeSize) {
		if status.Size, err = stateDBReadResult(ctx, m.state, func(ctx context.Context, client db.ReadOnly) (int, error) {
			return client.GetMailboxSize(ctx, m.snap.mboxID.InternalID)
		}); err != nil {
			return MailboxStatus{}, err
		}
	}

	return status, nil
}

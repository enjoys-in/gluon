package state

import (
	"context"
	"errors"
	"strings"

	"github.com/ProtonMail/gluon/db"
	"github.com/ProtonMail/gluon/imap"
	"github.com/ProtonMail/gluon/imap/command"
)

// GetACL returns the rights every identifier has on the mailbox.
func (state *State) GetACL(ctx context.Context, name string) ([]imap.ACLEntry, error) {
	mboxID, err := state.getACLMailboxID(ctx, name)
	if err != nil {
		return nil, err
	}

	entries, err := state.user.GetRemote().GetMailboxACL(ctx, mboxID)
	if err != nil {
		return nil, err
	}

	for i := range entries {
		entries[i].Rights = withObsoleteRights(entries[i].Rights)
	}

	return entries, nil
}

// SetACL changes the rights the identifier has on the mailbox. Rights are added to or removed from the ones the
// identifier has, or replace them, and the connector is given the result.
func (state *State) SetACL(ctx context.Context, name, identifier, rights string, mode command.ACLMode) error {
	mboxID, err := state.getACLMailboxID(ctx, name)
	if err != nil {
		return err
	}

	rights = withoutObsoleteRights(rights)

	if mode != command.ACLModeReplace {
		entries, err := state.user.GetRemote().GetMailboxACL(ctx, mboxID)
		if err != nil {
			return err
		}

		var current string

		for _, entry := range entries {
			if entry.Identifier == identifier {
				current = entry.Rights
			}
		}

		if mode == command.ACLModeAdd {
			rights = uniqueRights(current + rights)
		} else {
			rights = strings.Map(func(right rune) rune {
				if strings.ContainsRune(rights, right) {
					return -1
				}

				return right
			}, current)
		}
	}

	return state.user.GetRemote().SetMailboxACL(ctx, mboxID, identifier, rights)
}

// DeleteACL removes the rights the identifier has on the mailbox.
func (state *State) DeleteACL(ctx context.Context, name, identifier string) error {
	mboxID, err := state.getACLMailboxID(ctx, name)
	if err != nil {
		return err
	}

	return state.user.GetRemote().SetMailboxACL(ctx, mboxID, identifier, "")
}

// ListRights returns the rights the identifier always has on the mailbox and the rights that can be granted to it.
func (state *State) ListRights(ctx context.Context, name, identifier string) (string, []string, error) {
	mboxID, err := state.getACLMailboxID(ctx, name)
	if err != nil {
		return "", nil, err
	}

	return state.user.GetRemote().ListMailboxRights(ctx, mboxID, identifier)
}

// MyRights returns the rights the user has on the mailbox.
func (state *State) MyRights(ctx context.Context, name string) (string, error) {
	mboxID, err := state.getACLMailboxID(ctx, name)
	if err != nil {
		return "", err
	}

	rights, err := state.user.GetRemote().GetMyMailboxRights(ctx, mboxID)
	if err != nil {
		return "", err
	}

	return withObsoleteRights(rights), nil
}

// getACLMailboxID returns the connector's ID of the mailbox. The recovery mailbox only exists in Gluon and has no
// access control list.
func (state *State) getACLMailboxID(ctx context.Context, name string) (imap.MailboxID, error) {
	mbox, err := stateDBReadResult(ctx, state, func(ctx context.Context, client db.ReadOnly) (*db.Mailbox, error) {
		return client.GetMailboxByName(ctx, name)
	})
	if errors.Is(err, db.ErrNotFound) {
		return "", ErrNoSuchMailbox
	} else if err != nil {
		return "", err
	}

	if mbox.ID == state.user.GetRecoveryMailboxID().InternalID {
		return "", ErrOperationNotAllowed
	}

	return mbox.RemoteID, nil
}

// RFC 4314 keeps two rights of RFC 2086 for old clients: "c" stands for "k" and "x", and "d" for "e" and "t".
// The connector is only given the rights they stand for.
func withoutObsoleteRights(rights string) string {
	return uniqueRights(strings.NewReplacer("c", "kx", "d", "et").Replace(rights))
}

// withObsoleteRights adds "c" and "d" to rights that hold one of the rights they stand for, as RFC 4314 requires
// in ACL and MYRIGHTS responses.
func withObsoleteRights(rights string) string {
	if strings.ContainsAny(rights, "kx") {
		rights += "c"
	}

	if strings.ContainsAny(rights, "et") {
		rights += "d"
	}

	return uniqueRights(rights)
}

func uniqueRights(rights string) string {
	var unique strings.Builder

	for _, right := range rights {
		if !strings.ContainsRune(unique.String(), right) {
			unique.WriteRune(right)
		}
	}

	return unique.String()
}

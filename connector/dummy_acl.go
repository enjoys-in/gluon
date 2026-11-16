package connector

import (
	"context"
	"maps"
	"slices"
	"strings"

	"github.com/ProtonMail/gluon/imap"
)

// The dummy connector's user owns every mailbox: the owner has all rights, which cannot be changed, and can grant
// any right to other identifiers.

func (conn *Dummy) GetMailboxACL(_ context.Context, mboxID imap.MailboxID) ([]imap.ACLEntry, error) {
	conn.state.lock.RLock()
	defer conn.state.lock.RUnlock()

	mailbox, ok := conn.state.mailboxes[mboxID]
	if !ok {
		return nil, ErrNoSuchMailbox
	}

	entries := []imap.ACLEntry{{Identifier: conn.usernames[0], Rights: imap.ACLRights}}

	for _, identifier := range slices.Sorted(maps.Keys(mailbox.acl)) {
		entries = append(entries, imap.ACLEntry{Identifier: identifier, Rights: mailbox.acl[identifier]})
	}

	return entries, nil
}

func (conn *Dummy) SetMailboxACL(_ context.Context, mboxID imap.MailboxID, identifier string, rights string) error {
	conn.state.lock.Lock()
	defer conn.state.lock.Unlock()

	mailbox, ok := conn.state.mailboxes[mboxID]
	if !ok {
		return ErrNoSuchMailbox
	}

	if slices.Contains(conn.usernames, identifier) {
		return ErrOperationNotAllowed
	}

	if strings.Trim(rights, imap.ACLRights) != "" {
		return ErrOperationNotAllowed
	}

	if rights == "" {
		delete(mailbox.acl, identifier)

		return nil
	}

	if mailbox.acl == nil {
		mailbox.acl = make(map[string]string)
	}

	mailbox.acl[identifier] = rights

	return nil
}

func (conn *Dummy) ListMailboxRights(_ context.Context, mboxID imap.MailboxID, identifier string) (string, []string, error) {
	conn.state.lock.RLock()
	defer conn.state.lock.RUnlock()

	if _, ok := conn.state.mailboxes[mboxID]; !ok {
		return "", nil, ErrNoSuchMailbox
	}

	if slices.Contains(conn.usernames, identifier) {
		return imap.ACLRights, nil, nil
	}

	return "", strings.Split(imap.ACLRights, ""), nil
}

func (conn *Dummy) GetMyMailboxRights(_ context.Context, mboxID imap.MailboxID) (string, error) {
	conn.state.lock.RLock()
	defer conn.state.lock.RUnlock()

	if _, ok := conn.state.mailboxes[mboxID]; !ok {
		return "", ErrNoSuchMailbox
	}

	return imap.ACLRights, nil
}

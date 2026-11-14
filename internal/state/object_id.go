package state

import (
	"context"
	"errors"
	"strings"

	"github.com/ProtonMail/gluon/db"
	"github.com/ProtonMail/gluon/imap"
	"github.com/ProtonMail/gluon/imap/command"
	"github.com/ProtonMail/gluon/internal/response"
)

// mailboxObjectID returns the ID a client knows the mailbox by (RFC 8474). It is the connector's ID for the mailbox,
// so that it means the same on the connector's side. An ID that cannot be sent as an object ID is replaced by
// Gluon's own.
func mailboxObjectID(id db.MailboxIDPair) string {
	if imap.IsObjectID(string(id.RemoteID)) {
		return string(id.RemoteID)
	}

	return id.InternalID.String()
}

// messageObjectID returns the ID a client knows the message by, like mailboxObjectID.
func messageObjectID(id db.MessageIDPair) string {
	if imap.IsObjectID(string(id.RemoteID)) {
		return string(id.RemoteID)
	}

	return id.InternalID.String()
}

// ObjectID returns the mailbox's object ID.
func (m *Mailbox) ObjectID() string {
	return mailboxObjectID(m.id)
}

// MailboxObjectID returns the object ID of the mailbox with the given name.
func (state *State) MailboxObjectID(ctx context.Context, name string) (string, error) {
	// Like CREATE, accept a name that ends with the hierarchy separator.
	if strings.HasSuffix(name, state.delimiter) {
		name = strings.TrimRight(name, state.delimiter)
	}

	mbox, err := stateDBReadResult(ctx, state, func(ctx context.Context, client db.ReadOnly) (*db.Mailbox, error) {
		return client.GetMailboxByName(ctx, name)
	})
	if errors.Is(err, db.ErrNotFound) {
		return "", ErrNoSuchMailbox
	} else if err != nil {
		return "", err
	}

	return mailboxObjectID(db.NewMailboxIDPair(mbox)), nil
}

func fetchEmailID(msg snapMsgWithSeq, _ *db.Message, _ *fetchContext) (response.Item, error) {
	return response.ItemEmailID(messageObjectID(msg.ID)), nil
}

// fetchThreadID answers NIL: Gluon keeps no thread for a message.
func fetchThreadID(_ snapMsgWithSeq, _ *db.Message, _ *fetchContext) (response.Item, error) {
	return response.ItemThreadID(""), nil
}

func buildSearchOpEmailID(key *command.SearchKeyEmailID) (*buildSearchOpResult, error) {
	op := func(s *searchData) (bool, error) {
		return messageObjectID(s.message.ID) == key.Value, nil
	}

	return newBuildSearchOpResult(op), nil
}

// buildSearchOpThreadID matches nothing: no message has a thread ID.
func buildSearchOpThreadID() (*buildSearchOpResult, error) {
	op := func(*searchData) (bool, error) {
		return false, nil
	}

	return newBuildSearchOpResult(op), nil
}

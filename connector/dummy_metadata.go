package connector

import (
	"context"
	"maps"
	"slices"
	"strings"

	"github.com/ProtonMail/gluon/imap"
)

// The dummy connector keeps the least RFC 5464 asks a server to keep: ten annotations of 1024 bytes for each mailbox
// and for the server.
const (
	dummyMaxMetadataSize    = 1024
	dummyMaxMetadataEntries = 10
)

func (conn *Dummy) GetMetadata(_ context.Context, mboxID imap.MailboxID) ([]imap.MetadataEntry, error) {
	conn.state.lock.RLock()
	defer conn.state.lock.RUnlock()

	metadata, err := conn.state.getMetadata(mboxID)
	if err != nil {
		return nil, err
	}

	entries := make([]imap.MetadataEntry, 0, len(*metadata))

	for _, name := range slices.Sorted(maps.Keys(*metadata)) {
		entries = append(entries, imap.MetadataEntry{Name: name, Value: (*metadata)[name]})
	}

	return entries, nil
}

func (conn *Dummy) SetMetadata(_ context.Context, mboxID imap.MailboxID, entries []imap.MetadataEntry) error {
	conn.state.lock.Lock()
	defer conn.state.lock.Unlock()

	metadata, err := conn.state.getMetadata(mboxID)
	if err != nil {
		return err
	}

	// Every entry is checked on a copy first, so that a refused one changes nothing.
	updated := maps.Clone(*metadata)
	if updated == nil {
		updated = make(map[string][]byte)
	}

	for _, entry := range entries {
		switch {
		case entry.Value == nil:
			delete(updated, entry.Name)

		case len(entry.Value) > dummyMaxMetadataSize:
			return &MetadataTooBigError{MaxSize: dummyMaxMetadataSize}

		case conn.privateMetadataDisabled && strings.HasPrefix(entry.Name, "/private/"):
			return ErrMetadataNoPrivate

		default:
			updated[entry.Name] = entry.Value
		}
	}

	if len(updated) > dummyMaxMetadataEntries {
		return ErrMetadataTooMany
	}

	*metadata = updated

	return nil
}

// SetPrivateMetadataDisabled makes the connector refuse private annotations, as a server without them does.
func (conn *Dummy) SetPrivateMetadataDisabled(value bool) {
	conn.privateMetadataDisabled = value
}

// getMetadata returns the annotations of the mailbox, or of the server for the empty ID.
func (state *dummyState) getMetadata(mboxID imap.MailboxID) (*map[string][]byte, error) {
	if mboxID == "" {
		return &state.serverMetadata, nil
	}

	mailbox, ok := state.mailboxes[mboxID]
	if !ok {
		return nil, ErrNoSuchMailbox
	}

	return &mailbox.metadata, nil
}

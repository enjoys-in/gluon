package state

import (
	"context"
	"slices"
	"strings"

	"github.com/ProtonMail/gluon/imap"
	"github.com/ProtonMail/gluon/imap/command"
)

// GetMetadata returns the annotations a GETMETADATA command asks for: every requested entry and the entries below
// it down to the given depth. A requested entry that does not exist is returned with a nil value, unless entries
// below it were asked for too. Values larger than maxSize are left out when hasMaxSize is set; the size of the
// largest one left out is returned too.
func (state *State) GetMetadata(
	ctx context.Context,
	name string,
	entries []string,
	depth command.MetadataDepth,
	hasMaxSize bool,
	maxSize int,
) ([]imap.MetadataEntry, int, error) {
	mboxID, err := state.getMetadataMailboxID(ctx, name)
	if err != nil {
		return nil, 0, err
	}

	stored, err := state.user.GetRemote().GetMetadata(ctx, mboxID)
	if err != nil {
		return nil, 0, err
	}

	slices.SortFunc(stored, func(a, b imap.MetadataEntry) int {
		return strings.Compare(a.Name, b.Name)
	})

	var (
		result  []imap.MetadataEntry
		longest int
	)

	for _, requested := range entries {
		found := false

		for _, entry := range stored {
			below, isBelow := strings.CutPrefix(entry.Name, requested+"/")

			switch {
			case entry.Value == nil:
				continue

			case entry.Name == requested:
				found = true

			case !isBelow || depth == command.MetadataDepthZero:
				continue

			case depth == command.MetadataDepthOne && strings.Contains(below, "/"):
				continue
			}

			if hasMaxSize && len(entry.Value) > maxSize {
				longest = max(longest, len(entry.Value))
			} else {
				result = append(result, entry)
			}
		}

		if !found && depth == command.MetadataDepthZero {
			result = append(result, imap.MetadataEntry{Name: requested})
		}
	}

	return result, longest, nil
}

// SetMetadata changes annotations of the mailbox, or of the server when the name is empty.
func (state *State) SetMetadata(ctx context.Context, name string, entries []imap.MetadataEntry) error {
	mboxID, err := state.getMetadataMailboxID(ctx, name)
	if err != nil {
		return err
	}

	return state.user.GetRemote().SetMetadata(ctx, mboxID, entries)
}

// getMetadataMailboxID returns the connector's ID of the mailbox. The empty name stands for the server itself, which
// the connector knows by the empty ID.
func (state *State) getMetadataMailboxID(ctx context.Context, name string) (imap.MailboxID, error) {
	if name == "" {
		return "", nil
	}

	return state.getRemoteMailboxID(ctx, name)
}

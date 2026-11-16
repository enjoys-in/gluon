package session

import (
	"context"
	"errors"

	"github.com/ProtonMail/gluon/connector"
	"github.com/ProtonMail/gluon/imap/command"
	"github.com/ProtonMail/gluon/internal/response"
)

func (s *Session) handleGetMetadata(ctx context.Context, tag string, cmd *command.GetMetadata, ch chan response.Response) error {
	nameUTF8, err := s.decodeMetadataMailboxName(cmd.Mailbox)
	if err != nil {
		return err
	}

	entries, longest, err := s.state.GetMetadata(ctx, nameUTF8, cmd.Entries, cmd.Depth, cmd.HasMaxSize, cmd.MaxSize)
	if err != nil {
		return err
	}

	if len(entries) > 0 {
		ch <- response.Metadata(cmd.Mailbox, entries)
	}

	ok := response.Ok(tag).WithMessage("GETMETADATA")

	if longest > 0 {
		ok = ok.WithItems(response.ItemMetadataLongEntries(longest))
	}

	ch <- ok

	return nil
}

func (s *Session) handleSetMetadata(ctx context.Context, tag string, cmd *command.SetMetadata, ch chan response.Response) error {
	nameUTF8, err := s.decodeMetadataMailboxName(cmd.Mailbox)
	if err != nil {
		return err
	}

	var tooBig *connector.MetadataTooBigError

	if err := s.state.SetMetadata(ctx, nameUTF8, cmd.Entries); errors.As(err, &tooBig) {
		return response.No(tag).WithError(err).WithItems(response.ItemMetadataMaxSize(tooBig.MaxSize))
	} else if errors.Is(err, connector.ErrMetadataTooMany) {
		return response.No(tag).WithError(err).WithItems(response.ItemMetadataTooMany())
	} else if errors.Is(err, connector.ErrMetadataNoPrivate) {
		return response.No(tag).WithError(err).WithItems(response.ItemMetadataNoPrivate())
	} else if err != nil {
		return err
	}

	ch <- response.Ok(tag).WithMessage("SETMETADATA")

	return nil
}

// decodeMetadataMailboxName decodes the mailbox name of a METADATA command. The empty name is not a mailbox: it
// stands for the server.
func (s *Session) decodeMetadataMailboxName(name string) (string, error) {
	if name == "" {
		return "", nil
	}

	return s.decodeMailboxName(name)
}

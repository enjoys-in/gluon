package session

import (
	"context"

	"github.com/ProtonMail/gluon/imap/command"
	"github.com/ProtonMail/gluon/internal/response"
)

func (s *Session) handleSetACL(ctx context.Context, tag string, cmd *command.SetACL, ch chan response.Response) error {
	nameUTF8, err := s.decodeMailboxName(cmd.Mailbox)
	if err != nil {
		return err
	}

	if err := s.state.SetACL(ctx, nameUTF8, cmd.Identifier, cmd.Rights, cmd.Mode); err != nil {
		return err
	}

	ch <- response.Ok(tag).WithMessage("SETACL")

	return nil
}

func (s *Session) handleDeleteACL(ctx context.Context, tag string, cmd *command.DeleteACL, ch chan response.Response) error {
	nameUTF8, err := s.decodeMailboxName(cmd.Mailbox)
	if err != nil {
		return err
	}

	if err := s.state.DeleteACL(ctx, nameUTF8, cmd.Identifier); err != nil {
		return err
	}

	ch <- response.Ok(tag).WithMessage("DELETEACL")

	return nil
}

func (s *Session) handleGetACL(ctx context.Context, tag string, cmd *command.GetACL, ch chan response.Response) error {
	nameUTF8, err := s.decodeMailboxName(cmd.Mailbox)
	if err != nil {
		return err
	}

	entries, err := s.state.GetACL(ctx, nameUTF8)
	if err != nil {
		return err
	}

	ch <- response.ACL(cmd.Mailbox, entries)

	ch <- response.Ok(tag).WithMessage("GETACL")

	return nil
}

func (s *Session) handleListRights(ctx context.Context, tag string, cmd *command.ListRights, ch chan response.Response) error {
	nameUTF8, err := s.decodeMailboxName(cmd.Mailbox)
	if err != nil {
		return err
	}

	required, optional, err := s.state.ListRights(ctx, nameUTF8, cmd.Identifier)
	if err != nil {
		return err
	}

	ch <- response.ListRights(cmd.Mailbox, cmd.Identifier, required, optional)

	ch <- response.Ok(tag).WithMessage("LISTRIGHTS")

	return nil
}

func (s *Session) handleMyRights(ctx context.Context, tag string, cmd *command.MyRights, ch chan response.Response) error {
	nameUTF8, err := s.decodeMailboxName(cmd.Mailbox)
	if err != nil {
		return err
	}

	rights, err := s.state.MyRights(ctx, nameUTF8)
	if err != nil {
		return err
	}

	ch <- response.MyRights(cmd.Mailbox, rights)

	ch <- response.Ok(tag).WithMessage("MYRIGHTS")

	return nil
}

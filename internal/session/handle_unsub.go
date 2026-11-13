package session

import (
	"context"
	"errors"

	"github.com/ProtonMail/gluon/imap/command"
	"github.com/ProtonMail/gluon/internal/response"
	"github.com/ProtonMail/gluon/internal/state"
	"github.com/ProtonMail/gluon/profiling"
)

func (s *Session) handleUnsub(ctx context.Context, tag string, cmd *command.Unsubscribe, ch chan response.Response) error {
	profiling.Start(ctx, profiling.CmdTypeUnsubscribe)
	defer profiling.Stop(ctx, profiling.CmdTypeUnsubscribe)

	nameUTF8, err := s.decodeMailboxName(cmd.Mailbox)
	if err != nil {
		return err
	}

	if err := s.state.Unsubscribe(ctx, nameUTF8); err != nil && !errors.Is(err, state.ErrAlreadyUnsubscribed) {
		return err
	}

	ch <- response.Ok(tag).WithMessage("UNSUBSCRIBE")

	return nil
}

package session

import (
	"context"
	"errors"

	"github.com/ProtonMail/gluon/imap/command"
	"github.com/ProtonMail/gluon/internal/response"
	"github.com/ProtonMail/gluon/internal/state"
	"github.com/ProtonMail/gluon/profiling"
)

func (s *Session) handleSub(ctx context.Context, tag string, cmd *command.Subscribe, ch chan response.Response) error {
	profiling.Start(ctx, profiling.CmdTypeSubscribe)
	defer profiling.Stop(ctx, profiling.CmdTypeSubscribe)

	nameUTF8, err := s.decodeMailboxName(cmd.Mailbox)
	if err != nil {
		return err
	}

	// Already subscribed is success: clients such as Thunderbird subscribe right after CREATE.
	if err := s.state.Subscribe(ctx, nameUTF8); err != nil && !errors.Is(err, state.ErrAlreadySubscribed) {
		return err
	}

	ch <- response.Ok(tag).WithMessage("SUB")

	return nil
}

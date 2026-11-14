package session

import (
	"context"

	"github.com/ProtonMail/gluon/imap/command"
	"github.com/ProtonMail/gluon/internal/response"
)

func (s *Session) handleSetQuota(ctx context.Context, tag string, cmd *command.SetQuota, ch chan response.Response) error {
	root, err := s.state.SetQuota(ctx, cmd.Root, cmd.Limits)
	if err != nil {
		return err
	}

	ch <- response.Quota(root)

	ch <- response.Ok(tag).WithMessage("SETQUOTA")

	return nil
}

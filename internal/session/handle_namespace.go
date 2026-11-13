package session

import (
	"context"

	"github.com/ProtonMail/gluon/imap/command"
	"github.com/ProtonMail/gluon/internal/response"
)

func (s *Session) handleNamespace(_ context.Context, tag string, _ *command.Namespace, ch chan response.Response) error {
	ch <- response.Namespace(s.backend.GetDelimiter())

	ch <- response.Ok(tag).WithMessage("NAMESPACE")

	return nil
}

package session

import (
	"context"
	"errors"
	"strings"

	"github.com/ProtonMail/gluon/connector"
	"github.com/ProtonMail/gluon/imap"
	"github.com/ProtonMail/gluon/imap/command"
	"github.com/ProtonMail/gluon/internal/response"
	"github.com/ProtonMail/gluon/observability"
	"github.com/ProtonMail/gluon/observability/metrics"
	"github.com/ProtonMail/gluon/profiling"
)

func (s *Session) handleCreate(ctx context.Context, tag string, cmd *command.Create, ch chan response.Response) error {
	profiling.Start(ctx, profiling.CmdTypeCreate)
	defer profiling.Stop(ctx, profiling.CmdTypeCreate)

	nameUTF8, err := s.decodeMailboxName(cmd.Mailbox)
	if err != nil {
		return err
	}

	if strings.EqualFold(nameUTF8, imap.Inbox) {
		return ErrCreateInbox
	}

	// RFC 6154 CREATE-SPECIAL-USE Extension
	specialUse, err := validateSpecialUse(cmd.SpecialUse)
	if err != nil {
		return response.No(tag).WithError(err).WithItems(response.ItemUseAttr())
	}

	if err := s.state.CreateSpecialUse(ctx, nameUTF8, specialUse); errors.Is(err, connector.ErrSpecialUseNotSupported) {
		return response.No(tag).WithError(err).WithItems(response.ItemUseAttr())
	} else if err != nil {
		observability.AddMessageRelatedMetric(ctx, metrics.GenerateFailedToCreateMailbox())
		return err
	}

	ch <- response.Ok(tag).WithMessage("CREATE")

	return nil
}

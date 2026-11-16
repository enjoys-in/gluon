package session

import (
	"context"
	"errors"

	"github.com/ProtonMail/gluon/connector"
	"github.com/ProtonMail/gluon/imap/command"
	"github.com/ProtonMail/gluon/internal/contexts"
	"github.com/ProtonMail/gluon/internal/response"
	"github.com/ProtonMail/gluon/internal/state"
	"github.com/ProtonMail/gluon/observability"
	"github.com/ProtonMail/gluon/observability/metrics"
	"github.com/ProtonMail/gluon/profiling"
)

func (s *Session) handleCopy(ctx context.Context, tag string, cmd *command.Copy, mailbox *state.Mailbox, ch chan response.Response) (response.Response, error) {
	if contexts.IsUID(ctx) {
		profiling.Start(ctx, profiling.CmdTypeUIDCopy)
		defer profiling.Stop(ctx, profiling.CmdTypeUIDCopy)
	} else {
		profiling.Start(ctx, profiling.CmdTypeCopy)
		defer profiling.Stop(ctx, profiling.CmdTypeCopy)
	}

	// Due to how copies are handled, when the mailbox is read only we can't perform a copy as the message
	// will get moved.
	if mailbox.ReadOnly() {
		return nil, ErrReadOnly
	}

	nameUTF8, err := s.decodeMailboxName(cmd.Mailbox)
	if err != nil {
		return nil, err
	}

	// RFC 9738: a COPY is atomic, so one that is over the limit fails instead of copying some of the messages.
	if _, limit := s.limitSeqSet(ctx, mailbox, cmd.SeqSet, false); limit != nil {
		return response.No(tag).WithError(ErrTooManyMessages).WithItems(response.ItemMessageLimit(s.messageLimit, 0)), nil
	}

	item, err := mailbox.Copy(ctx, cmd.SeqSet, nameUTF8)
	if errors.Is(err, state.ErrNoSuchMessage) {
		return response.Bad(tag).WithError(err), nil
	} else if errors.Is(err, state.ErrNoSuchMailbox) {
		return response.No(tag).WithError(err).WithItems(response.ItemTryCreate()), nil
	} else if errors.Is(err, connector.ErrOverQuota) {
		return response.No(tag).WithError(err).WithItems(response.ItemOverQuota()), nil
	} else if err != nil {
		observability.AddMessageRelatedMetric(ctx, metrics.GenerateFailedToCopyMessagesMetric())
		return nil, err
	}

	response := response.Ok(tag)

	if item != nil {
		response = response.WithItems(item)
	}

	return response.WithMessage(okMessage(ctx)), nil
}

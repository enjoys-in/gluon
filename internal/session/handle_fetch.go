package session

import (
	"context"
	"errors"

	"github.com/ProtonMail/gluon/imap/command"
	"github.com/ProtonMail/gluon/internal/contexts"
	"github.com/ProtonMail/gluon/internal/response"
	"github.com/ProtonMail/gluon/internal/state"
	"github.com/ProtonMail/gluon/profiling"
	"github.com/ProtonMail/gluon/reporter"
)

func (s *Session) handleFetch(ctx context.Context, tag string, cmd *command.Fetch, mailbox *state.Mailbox, ch chan response.Response) (response.Response, error) {
	if contexts.IsUID(ctx) {
		profiling.Start(ctx, profiling.CmdTypeUIDFetch)
		defer profiling.Stop(ctx, profiling.CmdTypeUIDFetch)
	} else {
		profiling.Start(ctx, profiling.CmdTypeFetch)
		defer profiling.Stop(ctx, profiling.CmdTypeFetch)
	}

	limited := *cmd

	var limit response.Item

	limited.SeqSet, limit = s.limitSeqSet(ctx, mailbox, cmd.SeqSet, false)

	if err := mailbox.Fetch(ctx, &limited, ch); errors.Is(err, state.ErrNoSuchMessage) {
		return response.Bad(tag).WithError(err), nil
	} else if errors.Is(err, state.ErrUnknownCTE) {
		return nil, response.No(tag).WithError(err).WithItems(response.ItemUnknownCTE())
	} else if err != nil {
		if shouldReportIMAPCommandError(err) {
			// there's no events like this in sentry so far.
			reporter.MessageWithContext(ctx,
				"Failed to fetch messages",
				reporter.Context{"error": err},
			)
		}

		return nil, err
	}

	items := withItem(nil, limit)

	if mailbox.ExpungeIssued() {
		items = append(items, response.ItemExpungeIssued())
	}

	return response.Ok(tag).
		WithItems(items...).
		WithMessage(okMessage(ctx)), nil
}

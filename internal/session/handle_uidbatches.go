package session

import (
	"context"
	"fmt"

	"github.com/ProtonMail/gluon/imap/command"
	"github.com/ProtonMail/gluon/internal/response"
	"github.com/ProtonMail/gluon/internal/state"
)

const (
	// minUIDBatchSize is the smallest batch a client can ask for, so that a response stays a reasonable size.
	minUIDBatchSize = 10

	// maxUIDBatches is the most ranges one UIDBATCHES response carries.
	maxUIDBatches = 1000
)

var (
	ErrUIDBatchSizeTooSmall = fmt.Errorf("the batch size is smaller than %v", minUIDBatchSize)
	ErrTooManyUIDBatches    = fmt.Errorf("a response is limited to %v batches", maxUIDBatches)
)

func (s *Session) handleUIDBatches(ctx context.Context, tag string, cmd *command.UIDBatches, mailbox *state.Mailbox, ch chan response.Response) (response.Response, error) {
	if cmd.BatchSize < minUIDBatchSize {
		return nil, response.No(tag).WithError(ErrUIDBatchSizeTooSmall).WithItems(response.ItemTooFew())
	}

	first, last := 1, mailbox.UIDBatchCount(cmd.BatchSize)

	if cmd.HasRange() {
		first, last = cmd.From, cmd.To
	}

	if last-first+1 > maxUIDBatches {
		return nil, response.No(tag).WithError(ErrTooManyUIDBatches).WithItems(response.ItemTooMany())
	}

	res := response.UIDBatches(tag)

	for _, batch := range mailbox.UIDBatches(cmd.BatchSize, first, last) {
		res.WithRange(batch.High, batch.Low)
	}

	select {
	case ch <- res:

	case <-ctx.Done():
		return nil, ctx.Err()
	}

	return response.Ok(tag).WithMessage("UIDBATCHES"), nil
}

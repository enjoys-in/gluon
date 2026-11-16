package session

import (
	"context"

	"github.com/ProtonMail/gluon/imap/command"
	"github.com/ProtonMail/gluon/internal/contexts"
	"github.com/ProtonMail/gluon/internal/response"
	"github.com/ProtonMail/gluon/internal/state"
	"github.com/ProtonMail/gluon/profiling"
	"golang.org/x/text/encoding"
	"golang.org/x/text/encoding/ianaindex"
)

func (s *Session) handleSort(ctx context.Context, tag string, cmd *command.Sort, mailbox *state.Mailbox, ch chan response.Response) (response.Response, error) {
	if contexts.IsUID(ctx) {
		profiling.Start(ctx, profiling.CmdTypeUIDSort)
		defer profiling.Stop(ctx, profiling.CmdTypeUIDSort)
	} else {
		profiling.Start(ctx, profiling.CmdTypeSort)
		defer profiling.Stop(ctx, profiling.CmdTypeSort)
	}

	var decoder *encoding.Decoder

	if len(cmd.Charset) != 0 {
		enc, err := ianaindex.IANA.Encoding(cmd.Charset)
		if err != nil {
			return nil, response.No(tag).WithItems(response.ItemBadCharset())
		}

		decoder = enc.NewDecoder()
	} else {
		decoder = encoding.Nop.NewDecoder()
	}

	seq, err := mailbox.Sort(ctx, cmd.SortKeys, cmd.Keys, decoder)
	if err != nil {
		return nil, err
	}

	select {
	case ch <- newSortResponse(ctx, tag, cmd, seq):

	case <-ctx.Done():
		return nil, ctx.Err()
	}

	var items []response.Item

	if mailbox.ExpungeIssued() {
		items = append(items, response.ItemExpungeIssued())
	}

	return response.Ok(tag).
		WithItems(items...).
		WithMessage(okMessage(ctx)), nil
}

func newSortResponse(ctx context.Context, tag string, cmd *command.Sort, seq []uint32) response.Response {
	if len(cmd.Return) == 0 {
		return response.Sort(seq...)
	}

	// RFC5267 ESORT Extension
	return withSearchReturnOptions(response.ESort(tag, contexts.IsUID(ctx), seq...), cmd.Return)
}

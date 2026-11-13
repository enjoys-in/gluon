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

func (s *Session) handleThread(ctx context.Context, tag string, cmd *command.Thread, mailbox *state.Mailbox, ch chan response.Response) (response.Response, error) {
	if contexts.IsUID(ctx) {
		profiling.Start(ctx, profiling.CmdTypeUIDThread)
		defer profiling.Stop(ctx, profiling.CmdTypeUIDThread)
	} else {
		profiling.Start(ctx, profiling.CmdTypeThread)
		defer profiling.Stop(ctx, profiling.CmdTypeThread)
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

	threads, err := mailbox.Thread(ctx, cmd.Algorithm, cmd.Keys, decoder)
	if err != nil {
		return nil, err
	}

	select {
	case ch <- response.Thread(threads):

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

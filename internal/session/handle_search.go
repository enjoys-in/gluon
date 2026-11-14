package session

import (
	"context"
	"slices"

	"github.com/ProtonMail/gluon/imap/command"
	"github.com/ProtonMail/gluon/internal/contexts"
	"github.com/ProtonMail/gluon/internal/response"
	"github.com/ProtonMail/gluon/internal/state"
	"github.com/ProtonMail/gluon/profiling"
	"golang.org/x/text/encoding"
	"golang.org/x/text/encoding/ianaindex"
)

func (s *Session) handleSearch(ctx context.Context, tag string, cmd *command.Search, mailbox *state.Mailbox, ch chan response.Response) (response.Response, error) {
	if contexts.IsUID(ctx) {
		profiling.Start(ctx, profiling.CmdTypeUIDSearch)
		defer profiling.Stop(ctx, profiling.CmdTypeUIDSearch)
	} else {
		profiling.Start(ctx, profiling.CmdTypeSearch)
		defer profiling.Stop(ctx, profiling.CmdTypeSearch)
	}

	var decoder *encoding.Decoder

	if len(cmd.Charset) != 0 {
		encoding, err := ianaindex.IANA.Encoding(cmd.Charset)
		if err != nil {
			return nil, response.No(tag).WithItems(response.ItemBadCharset())
		}

		decoder = encoding.NewDecoder()
	} else {
		decoder = encoding.Nop.NewDecoder()
	}

	seq, err := search(ctx, cmd, mailbox, decoder)
	if err != nil {
		return nil, err
	}

	if res := newSearchResponse(ctx, tag, cmd, seq); res != nil {
		select {
		case ch <- res:

		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}

	var items []response.Item

	if mailbox.ExpungeIssued() {
		items = append(items, response.ItemExpungeIssued())
	}

	return response.Ok(tag).
		WithItems(items...).
		WithMessage(okMessage(ctx)), nil
}

func search(ctx context.Context, cmd *command.Search, mailbox *state.Mailbox, decoder *encoding.Decoder) ([]uint32, error) {
	seq, err := searchMessages(ctx, cmd, mailbox, decoder)

	// RFC 5182 SEARCHRES Extension: a search that fails leaves an empty result behind.
	if slices.Contains(cmd.Return, command.SearchReturnSave) {
		mailbox.SaveSearchResult(ctx, searchResultToSave(cmd.Return, seq))
	}

	return seq, err
}

func searchMessages(ctx context.Context, cmd *command.Search, mailbox *state.Mailbox, decoder *encoding.Decoder) ([]uint32, error) {
	if seq, ok, err := mailbox.SearchWithConnector(ctx, cmd.Keys, decoder); err != nil || ok {
		return seq, err
	}

	return mailbox.Search(ctx, cmd.Keys, decoder)
}

// searchResultToSave returns what "$" stands for after the search: every message found, or only the lowest and the
// highest one when the command asks for nothing but MIN and MAX.
func searchResultToSave(opts []command.SearchReturnOption, seq []uint32) []uint32 {
	if len(seq) == 0 || slices.Contains(opts, command.SearchReturnAll) || slices.Contains(opts, command.SearchReturnCount) {
		return seq
	}

	lowest, highest := slices.Min(seq), slices.Max(seq)

	switch hasMin, hasMax := slices.Contains(opts, command.SearchReturnMin), slices.Contains(opts, command.SearchReturnMax); {
	case hasMin && hasMax && lowest != highest:
		return []uint32{lowest, highest}

	case hasMin:
		return []uint32{lowest}

	case hasMax:
		return []uint32{highest}

	default:
		return seq
	}
}

func newSearchResponse(ctx context.Context, tag string, cmd *command.Search, seq []uint32) response.Response {
	if len(cmd.Return) == 0 {
		return response.Search(seq...)
	}

	// RFC 5182 SEARCHRES Extension: SAVE on its own sends no search result.
	if len(cmd.Return) == 1 && cmd.Return[0] == command.SearchReturnSave {
		return nil
	}

	// RFC4731 ESEARCH Extension
	return withSearchReturnOptions(response.ESearch(tag, contexts.IsUID(ctx), seq...), cmd.Return)
}

func withSearchReturnOptions[T interface {
	WithMin() T
	WithMax() T
	WithAll() T
	WithCount() T
}](res T, opts []command.SearchReturnOption) T {
	for _, opt := range opts {
		switch opt {
		case command.SearchReturnMin:
			res.WithMin()
		case command.SearchReturnMax:
			res.WithMax()
		case command.SearchReturnAll:
			res.WithAll()
		case command.SearchReturnCount:
			res.WithCount()
		}
	}

	return res
}

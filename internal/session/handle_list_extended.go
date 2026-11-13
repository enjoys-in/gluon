package session

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/ProtonMail/gluon/imap"
	"github.com/ProtonMail/gluon/imap/command"
	"github.com/ProtonMail/gluon/internal/response"
	"github.com/ProtonMail/gluon/internal/state"
	"github.com/emersion/go-imap/utf7"
)

var listStatusSkippedAttributes = []string{strings.ToLower(imap.AttrNoSelect), strings.ToLower(imap.AttrNonExistent)}

func (s *Session) handleListExtended(ctx context.Context, tag string, cmd *command.List, ch chan response.Response) error {
	patterns := make([]string, 0, len(cmd.Extended.MoreMailboxes)+1)

	for _, pattern := range cmd.Patterns() {
		nameUTF8, err := s.decodeMailboxName(pattern)
		if err != nil {
			return err
		}

		patterns = append(patterns, nameUTF8)
	}

	var matches []state.Match

	if err := s.state.ListExtended(ctx, cmd.Mailbox, patterns, state.ListOptions{
		SelectSubscribed:     cmd.Extended.Select.Subscribed,
		SelectRecursiveMatch: cmd.Extended.Select.RecursiveMatch,
		SelectSpecialUse:     cmd.Extended.Select.SpecialUse,
		ReturnSubscribed:     cmd.Extended.Return.Subscribed,
	}, func(result []state.Match) error {
		matches = result
		return nil
	}); err != nil {
		return err
	}

	encoder := utf7.Encoding.NewEncoder()

	for _, match := range matches {
		nameUtf7, err := encoder.String(match.Name)
		if err != nil {
			return fmt.Errorf("failed to convert name to utf7")
		}

		select {
		case ch <- response.List().
			WithName(nameUtf7).
			WithDelimiter(match.Delimiter).
			WithAttributes(match.Atts).
			WithChildInfo(match.ChildInfo):

		case <-ctx.Done():
			return ctx.Err()
		}

		// RFC5819 LIST-STATUS Extension
		if len(cmd.Extended.Return.Status) == 0 || match.Atts.ContainsAnyUnchecked(listStatusSkippedAttributes...) {
			continue
		}

		items, err := s.statusItems(ctx, match.Name, cmd.Extended.Return.Status, ch)
		if errors.Is(err, state.ErrNoSuchMailbox) {
			continue
		} else if err != nil {
			return err
		}

		ch <- response.Status().WithMailbox(nameUtf7).WithItems(items...)
	}

	ch <- response.Ok(tag).WithMessage("LIST")

	return nil
}

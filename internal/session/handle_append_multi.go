package session

import (
	"context"
	"errors"

	"github.com/ProtonMail/gluon/connector"
	"github.com/ProtonMail/gluon/imap/command"
	"github.com/ProtonMail/gluon/internal/response"
	"github.com/ProtonMail/gluon/internal/state"
	"github.com/ProtonMail/gluon/rfcvalidation"
)

// handleMultiAppend handles an APPEND command with several messages (RFC 3502). Every message is checked before
// any of them is appended.
func (s *Session) handleMultiAppend(ctx context.Context, tag string, cmd *command.Append, nameUTF8 string, ch chan response.Response) error {
	messages := make([]connector.MessageToCreate, 0, len(cmd.More)+1)

	for _, message := range cmd.Messages() {
		flags, err := validateStoreFlags(message.Flags)
		if err != nil {
			return response.Bad(tag).WithError(err)
		}

		messages = append(messages, connector.MessageToCreate{Literal: message.Literal, Flags: flags, Date: message.DateTime})
	}

	if err := s.state.AppendOnlyMailbox(ctx, nameUTF8, func(mailbox state.AppendOnlyMailbox, isSameMBox bool) error {
		isDrafts, err := mailbox.IsDrafts(ctx)
		if err != nil {
			return err
		}

		if !isDrafts {
			for _, message := range messages {
				if err := rfcvalidation.ValidateMessageHeaderFields(message.Literal); err != nil {
					return response.Bad(tag).WithError(err)
				}
			}
		}

		messageUIDs, err := mailbox.AppendAll(ctx, messages)
		if err != nil {
			return err
		}

		if isSameMBox {
			if err := flush(ctx, mailbox, true, ch); err != nil {
				return err
			}
		}

		ch <- response.Ok(tag).WithItems(response.ItemAppendUIDs(mailbox.UIDValidity(), messageUIDs)).WithMessage("APPEND")

		return nil
	}); errors.Is(err, state.ErrNoSuchMailbox) {
		return response.No(tag).WithError(err).WithItems(response.ItemTryCreate())
	} else if errors.Is(err, connector.ErrOverQuota) {
		return response.No(tag).WithError(err).WithItems(response.ItemOverQuota())
	} else if err != nil {
		return err
	}

	return nil
}

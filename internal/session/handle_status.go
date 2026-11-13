package session

import (
	"context"

	"github.com/ProtonMail/gluon/imap/command"
	"github.com/ProtonMail/gluon/internal/response"
	"github.com/ProtonMail/gluon/internal/state"
	"github.com/ProtonMail/gluon/profiling"
)

func (s *Session) handleStatus(ctx context.Context, tag string, cmd *command.Status, ch chan response.Response) error {
	profiling.Start(ctx, profiling.CmdTypeStatus)
	defer profiling.Stop(ctx, profiling.CmdTypeStatus)

	nameUTF8, err := s.decodeMailboxName(cmd.Mailbox)
	if err != nil {
		return err
	}

	items, err := s.statusItems(ctx, nameUTF8, cmd.Attributes, ch)
	if err != nil {
		return err
	}

	ch <- response.Status().WithMailbox(cmd.Mailbox).WithItems(items...)

	ch <- response.Ok(tag).WithMessage("STATUS")

	return nil
}

func (s *Session) statusItems(
	ctx context.Context,
	nameUTF8 string,
	attributes []command.StatusAttribute,
	ch chan response.Response,
) ([]response.Item, error) {
	status, ok, err := s.state.MailboxStatus(ctx, nameUTF8, attributes)
	if err != nil {
		return nil, err
	}

	if !ok {
		if err := s.state.Mailbox(ctx, nameUTF8, func(mailbox *state.Mailbox) error {
			if mailbox.Selected() {
				if err := flush(ctx, mailbox, true, ch); err != nil {
					return err
				}
			}

			status, err = mailbox.Status(ctx)

			return err
		}); err != nil {
			return nil, err
		}
	}

	items := make([]response.Item, 0, len(attributes))

	for _, att := range attributes {
		switch att {
		case command.StatusAttributeMessages:
			items = append(items, response.ItemMessages(status.Messages))

		case command.StatusAttributeRecent:
			items = append(items, response.ItemRecent(status.Recent))

		case command.StatusAttributeUIDNext:
			items = append(items, response.ItemUIDNext(status.UIDNext))

		case command.StatusAttributeUIDValidity:
			items = append(items, response.ItemUIDValidity(status.UIDValidity))

		case command.StatusAttributeUnseen:
			items = append(items, response.ItemUnseen(uint32(status.Unseen)))
		}
	}

	return items, nil
}

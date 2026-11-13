package state

import (
	"context"
	"errors"
	"fmt"

	"github.com/ProtonMail/gluon/connector"
	"github.com/ProtonMail/gluon/db"
	"github.com/ProtonMail/gluon/imap"
	"github.com/ProtonMail/gluon/internal/ids"
	"github.com/ProtonMail/gluon/rfc822"
)

// AppendAll appends the messages of one APPEND command and returns their UIDs. The connector creates all of them or
// none and Gluon adds them in one transaction, so the mailbox never ends up with only some of them (RFC 3502).
func (m *Mailbox) AppendAll(ctx context.Context, messages []connector.MessageToCreate) ([]imap.UID, error) {
	uids, err := m.appendAllRegular(ctx, messages)
	if err != nil && !errors.Is(err, connector.ErrMessageSizeExceedsLimits) {
		// Like a single APPEND, the messages that could not be appended are kept in the recovery mailbox.
		for _, message := range messages {
			knownMessage, recoverErr := stateDBWriteResult(ctx, m.state, func(ctx context.Context, tx db.Transaction) ([]Update, bool, error) {
				return m.state.actionCreateRecoveredMessage(ctx, tx, message.Literal, message.Flags, message.Date)
			})
			if recoverErr != nil && !knownMessage {
				m.log.WithError(recoverErr).Error("Failed to insert message into recovery mailbox")
			}

			if knownMessage && !errors.Is(err, ErrKnownRecoveredMessage) {
				err = fmt.Errorf("%v: %w", err, ErrKnownRecoveredMessage)
			}
		}
	}

	return uids, err
}

func (m *Mailbox) appendAllRegular(ctx context.Context, messages []connector.MessageToCreate) ([]imap.UID, error) {
	appendIntoDrafts, err := m.IsDrafts(ctx)
	if err != nil {
		return nil, err
	}

	// Force create messages when appending to drafts so that IMAP clients can create new draft messages.
	if appendIntoDrafts {
		drafts := make([]connector.MessageToCreate, 0, len(messages))

		for _, message := range messages {
			if literal, err := rfc822.EraseHeaderValue(message.Literal, ids.InternalIDKey); err != nil {
				m.log.WithError(err).Error("Failed to erase Gluon internal id from draft")
			} else {
				message.Literal = literal
			}

			drafts = append(drafts, message)
		}

		messages = drafts
	}

	return stateDBWriteResult(ctx, m.state, func(ctx context.Context, tx db.Transaction) ([]Update, []imap.UID, error) {
		messageCount, uid, err := tx.GetMailboxMessageCountAndUID(ctx, m.snap.mboxID.InternalID)
		if err != nil {
			return nil, nil, err
		}

		if err := m.state.imapLimits.CheckMailBoxMessageCount(messageCount, len(messages)); err != nil {
			return nil, nil, err
		}

		if err := m.state.imapLimits.CheckUIDCount(uid, len(messages)); err != nil {
			return nil, nil, err
		}

		updates, internalIDs, created, literals, err := m.state.user.GetRemote().CreateMessages(ctx, tx, m.snap.mboxID.RemoteID, messages)
		if err != nil {
			return nil, nil, err
		}

		uids := make([]imap.UID, 0, len(created))

		for i := range created {
			var messageUID imap.UID

			updates, messageUID, err = m.state.actionAddCreatedMessage(
				ctx,
				tx,
				m.snap.mboxID,
				updates,
				internalIDs[i],
				created[i],
				literals[i],
				m.snap == m.state.snap,
				appendIntoDrafts,
			)
			if err != nil {
				return nil, nil, err
			}

			uids = append(uids, messageUID)
		}

		return updates, uids, nil
	})
}

package connector

import (
	"context"

	"github.com/ProtonMail/gluon/imap"
)

func (conn *Dummy) CreateMessages(ctx context.Context, _ IMAPStateWrite, mboxID imap.MailboxID, messages []MessageToCreate) ([]imap.Message, [][]byte, error) {
	conn.state.recordIMAPID(ctx)

	// Every message is parsed before the first one is created, so that a bad one leaves nothing behind.
	parsed := make([]*imap.ParsedMessage, 0, len(messages))

	for _, message := range messages {
		parsedMessage, err := imap.NewParsedMessage(message.Literal)
		if err != nil {
			return nil, nil, err
		}

		parsed = append(parsed, parsedMessage)
	}

	created := make([]imap.Message, 0, len(messages))
	literals := make([][]byte, 0, len(messages))
	updates := make([]*imap.MessageCreated, 0, len(messages))

	for i, message := range messages {
		createdMessage := conn.state.createMessage(
			mboxID,
			message.Literal,
			parsed[i],
			message.Flags.ContainsUnchecked(imap.FlagSeenLowerCase),
			message.Flags.ContainsUnchecked(imap.FlagFlaggedLowerCase),
			message.Flags,
			message.Date,
		)

		created = append(created, createdMessage)
		literals = append(literals, message.Literal)
		updates = append(updates, &imap.MessageCreated{
			Message:       createdMessage,
			Literal:       message.Literal,
			MailboxIDs:    []imap.MailboxID{mboxID},
			ParsedMessage: parsed[i],
		})
	}

	conn.pushUpdate(imap.NewMessagesCreated(conn.allowMessageCreateWithUnknownMailboxID, updates...))

	return created, literals, nil
}

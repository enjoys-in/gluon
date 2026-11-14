package state

import (
	"context"
	"time"

	"github.com/ProtonMail/gluon/connector"
	"github.com/ProtonMail/gluon/db"
	"github.com/ProtonMail/gluon/imap"
	"github.com/ProtonMail/gluon/imap/command"
)

// Connector interface for State differs slightly from the connector.Connector interface as it needs the ability
// to generate internal IDs for each request as well as track local metadata associated with each state. The local
// metadata (e.g.: IMAP ID extension info) should be injected into the context before each call to ensure the
// connector.Connector can receive this information.
// Sadly, due to Go's cyclic dependencies, this needs to be an interface. The implementation of this interface
// is available in the backend package.
type Connector interface {
	// SetConnMetadataValue sets a metadata value associated with the current connector.
	SetConnMetadataValue(key string, value any)

	// ClearConnMetadataValue clears a metadata value associated with the current connector.
	ClearConnMetadataValue(key string)

	// ClearAllConnMetadata clears all metadata values associated with the current connector.
	ClearAllConnMetadata()

	// CreateMailbox creates a new mailbox with the given name.
	CreateMailbox(ctx context.Context, tx db.Transaction, name []string) ([]Update, imap.Mailbox, error)

	// CreateSpecialUseMailbox creates a new mailbox with the given name and special-use attributes.
	CreateSpecialUseMailbox(ctx context.Context, tx db.Transaction, name []string, attributes imap.FlagSet) ([]Update, imap.Mailbox, error)

	// UpdateMailbox sets the name of the mailbox with the given ID to the given new name.
	UpdateMailbox(ctx context.Context, tx db.Transaction, mboxID imap.MailboxID, newName []string) ([]Update, error)

	// DeleteMailbox deletes the mailbox with the given ID and name.
	DeleteMailbox(ctx context.Context, tx db.Transaction, mboxID imap.MailboxID) ([]Update, error)

	// SetMailboxSubscribed sets whether the mailbox with the given ID is subscribed.
	SetMailboxSubscribed(ctx context.Context, tx db.Transaction, mboxID imap.MailboxID, subscribed bool) ([]Update, error)

	// CreateMessage appends a message literal to the mailbox with the given ID.
	CreateMessage(
		ctx context.Context,
		tx db.Transaction,
		mboxID imap.MailboxID,
		literal []byte,
		flags imap.FlagSet,
		date time.Time,
	) ([]Update, imap.InternalMessageID, imap.Message, []byte, error)

	// CreateMessages appends several message literals to the mailbox with the given ID. The connector creates all of
	// them or none.
	CreateMessages(
		ctx context.Context,
		tx db.Transaction,
		mboxID imap.MailboxID,
		messages []connector.MessageToCreate,
	) ([]Update, []imap.InternalMessageID, []imap.Message, [][]byte, error)

	// GetMessageLiteral retrieves the message literal from the connector.
	// Note: this can get called from different go routines.
	GetMessageLiteral(ctx context.Context, id imap.MessageID) ([]byte, error)

	// AddMessagesToMailbox adds the message with the given ID to the mailbox with the given ID.
	AddMessagesToMailbox(
		ctx context.Context,
		tx db.Transaction,
		messageIDs []imap.MessageID,
		mboxID imap.MailboxID,
	) ([]Update, error)

	// RemoveMessagesFromMailbox removes the message with the given ID from the mailbox with the given ID.
	RemoveMessagesFromMailbox(
		ctx context.Context,
		tx db.Transaction,
		messageIDs []imap.MessageID,
		mboxID imap.MailboxID,
	) ([]Update, error)

	// MoveMessagesFromMailbox removes the message with the given ID from the mailbox with the given ID.
	MoveMessagesFromMailbox(
		ctx context.Context,
		tx db.Transaction,
		messageIDs []imap.MessageID,
		mboxFromID imap.MailboxID,
		mboxToID imap.MailboxID,
	) ([]Update, bool, error)

	// SetMessagesSeen marks the message with the given ID as seen or unseen.
	SetMessagesSeen(ctx context.Context, tx db.Transaction, messageIDs []imap.MessageID, seen bool) ([]Update, error)

	// SetMessagesFlagged marks the message with the given ID as seen or unseen.
	SetMessagesFlagged(ctx context.Context, tx db.Transaction, messageIDs []imap.MessageID, flagged bool) ([]Update, error)

	// GetMailboxVisibility retrieves the visibility status of a mailbox for a client.
	GetMailboxVisibility(ctx context.Context, id imap.MailboxID) imap.MailboxVisibility

	// SetMessagesForwarded marks the message with the given ID as forwarded.
	SetMessagesForwarded(ctx context.Context, tx db.Transaction, messageIDs []imap.MessageID, forwarded bool) ([]Update, error)

	// GetQuota returns the quota for the given quota root name.
	GetQuota(ctx context.Context, rootName string) (*imap.QuotaRoot, error)

	// GetQuotaRoot returns the list of quota root names applicable to the given mailbox,
	// along with the resource usage for each of those roots.
	GetQuotaRoot(ctx context.Context, mailboxName string) ([]string, []*imap.QuotaRoot, error)

	// SetQuotaLimits changes limits of the quota root with the given name and returns the root as it is afterwards.
	SetQuotaLimits(ctx context.Context, rootName string, limits []imap.QuotaLimit) (*imap.QuotaRoot, error)

	// SearchMessages returns the messages in the mailbox that match every key, or false when the connector
	// leaves the search to Gluon.
	SearchMessages(ctx context.Context, mboxID imap.MailboxID, keys []command.SearchKey) ([]imap.MessageID, bool, error)
}

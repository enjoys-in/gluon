package backend

import (
	"context"

	"github.com/ProtonMail/gluon/db"
	"github.com/ProtonMail/gluon/imap"
)

// applyMailboxSubscribed applies a MailboxSubscribed update.
func (user *user) applyMailboxSubscribed(ctx context.Context, update *imap.MailboxSubscribed) error {
	return user.db.Write(ctx, func(ctx context.Context, tx db.Transaction) error {
		mboxID, err := tx.GetMailboxIDFromRemoteID(ctx, update.MailboxID)
		if db.IsErrNotFound(err) {
			return nil
		} else if err != nil {
			return err
		}

		return tx.SetMailboxSubscribed(ctx, mboxID, update.Subscribed)
	})
}

package imap

import "fmt"

type MailboxSubscribed struct {
	updateBase

	*updateWaiter

	MailboxID  MailboxID
	Subscribed bool
}

func NewMailboxSubscribed(mailboxID MailboxID, subscribed bool) *MailboxSubscribed {
	return &MailboxSubscribed{
		updateWaiter: newUpdateWaiter(),
		MailboxID:    mailboxID,
		Subscribed:   subscribed,
	}
}

func (u *MailboxSubscribed) String() string {
	return fmt.Sprintf(
		"MailboxSubscribed: MailboxID = %v, Subscribed = %v",
		u.MailboxID.ShortID(),
		u.Subscribed,
	)
}

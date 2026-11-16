package tests

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/ProtonMail/gluon/connector"
	"github.com/ProtonMail/gluon/imap"
	"github.com/stretchr/testify/require"
)

type subscription struct {
	mboxID     imap.MailboxID
	subscribed bool
}

// subscribeConnector records the subscriptions Gluon hands over and can reject them.
type subscribeConnector struct {
	*connector.Dummy

	lock   sync.Mutex
	calls  []subscription
	reject error
}

func (c *subscribeConnector) SetMailboxSubscribed(ctx context.Context, cache connector.IMAPStateWrite, mboxID imap.MailboxID, subscribed bool) error {
	c.lock.Lock()
	c.calls = append(c.calls, subscription{mboxID: mboxID, subscribed: subscribed})
	reject := c.reject
	c.lock.Unlock()

	if reject != nil {
		return reject
	}

	return c.Dummy.SetMailboxSubscribed(ctx, cache, mboxID, subscribed)
}

func (c *subscribeConnector) setReject(err error) {
	c.lock.Lock()
	defer c.lock.Unlock()

	c.reject = err
}

func (c *subscribeConnector) subscriptions() []subscription {
	c.lock.Lock()
	defer c.lock.Unlock()

	return c.calls
}

type subscribeConnectorBuilder struct {
	conn *subscribeConnector
}

func (b *subscribeConnectorBuilder) New(usernames []string, password []byte, period time.Duration, flags, permFlags, attrs imap.FlagSet) Connector {
	b.conn = &subscribeConnector{Dummy: connector.NewDummy(usernames, password, period, flags, permFlags, attrs)}

	return b.conn
}

func runSubscribeConnectorTest(t *testing.T, tests func(c *testConnection, s *testSession, conn *subscribeConnector, mboxID imap.MailboxID)) {
	builder := &subscribeConnectorBuilder{}

	runOneToOneTestWithAuth(t, defaultServerOptions(t, withConnectorBuilder(builder)), func(c *testConnection, s *testSession) {
		mboxID := s.mailboxCreated("user", []string{"Work"})

		s.flush("user")

		tests(c, s, builder.conn, mboxID)
	})
}

func TestSubscribeTellsTheConnector(t *testing.T) {
	runSubscribeConnectorTest(t, func(c *testConnection, s *testSession, conn *subscribeConnector, mboxID imap.MailboxID) {
		require.True(t, s.isMailboxSubscribed("user", mboxID))

		c.C(`A001 UNSUBSCRIBE Work`).OK(`A001`)
		require.False(t, s.isMailboxSubscribed("user", mboxID))

		c.C(`A002 LSUB "" "*"`).S(`* LSUB (\Unmarked) "/" "INBOX"`).OK(`A002`)

		c.C(`A003 SUBSCRIBE Work`).OK(`A003`)
		require.True(t, s.isMailboxSubscribed("user", mboxID))

		c.C(`A004 LSUB "" "*"`).S(`* LSUB (\Unmarked) "/" "INBOX"`, `* LSUB (\Unmarked) "/" "Work"`).OK(`A004`)

		require.Equal(t, []subscription{{mboxID: mboxID, subscribed: false}, {mboxID: mboxID, subscribed: true}}, conn.subscriptions())
	})
}

func TestSubscribeDoesNotTellTheConnectorWhenNothingChanges(t *testing.T) {
	runSubscribeConnectorTest(t, func(c *testConnection, _ *testSession, conn *subscribeConnector, mboxID imap.MailboxID) {
		c.C(`A001 SUBSCRIBE Work`).OK(`A001`)
		c.C(`A002 SUBSCRIBE Missing`).NO(`A002`)
		c.C(`A003 UNSUBSCRIBE Missing`).NO(`A003`)

		require.Empty(t, conn.subscriptions())

		c.C(`A004 UNSUBSCRIBE Work`).OK(`A004`)
		c.C(`A005 UNSUBSCRIBE Work`).OK(`A005`)

		require.Equal(t, []subscription{{mboxID: mboxID, subscribed: false}}, conn.subscriptions())
	})
}

func TestSubscribeConnectorErrorRejectsTheChange(t *testing.T) {
	runSubscribeConnectorTest(t, func(c *testConnection, _ *testSession, conn *subscribeConnector, _ imap.MailboxID) {
		conn.setReject(errors.New("subscriptions are read only"))

		c.C(`A001 UNSUBSCRIBE Work`)
		c.Sx(`^A001 NO .*subscriptions are read only`)

		c.C(`A002 LSUB "" "*"`).S(`* LSUB (\Unmarked) "/" "INBOX"`, `* LSUB (\Unmarked) "/" "Work"`).OK(`A002`)

		conn.setReject(nil)

		c.C(`A003 UNSUBSCRIBE Work`).OK(`A003`)

		conn.setReject(errors.New("subscriptions are read only"))

		c.C(`A004 SUBSCRIBE Work`)
		c.Sx(`^A004 NO .*subscriptions are read only`)

		c.C(`A005 LSUB "" "*"`).S(`* LSUB (\Unmarked) "/" "INBOX"`).OK(`A005`)
	})
}

func TestSubscribeConnectorUpdateChangesTheSubscription(t *testing.T) {
	runSubscribeConnectorTest(t, func(c *testConnection, s *testSession, conn *subscribeConnector, mboxID imap.MailboxID) {
		s.mailboxSubscribed("user", mboxID, false)

		c.C(`A001 LSUB "" "*"`).S(`* LSUB (\Unmarked) "/" "INBOX"`).OK(`A001`)
		c.C(`A002 LIST (SUBSCRIBED) "" "*"`).S(`* LIST (\HasNoChildren \Subscribed \Unmarked) "/" "INBOX"`).OK(`A002`)

		s.mailboxSubscribed("user", mboxID, true)

		c.C(`A003 LSUB "" "*"`).S(`* LSUB (\Unmarked) "/" "INBOX"`, `* LSUB (\Unmarked) "/" "Work"`).OK(`A003`)

		// A change the connector reports is not sent back to it.
		require.Empty(t, conn.subscriptions())
	})
}

func TestSubscribeDeletedMailboxStaysInGluon(t *testing.T) {
	runSubscribeConnectorTest(t, func(c *testConnection, _ *testSession, conn *subscribeConnector, _ imap.MailboxID) {
		c.C(`A001 DELETE Work`).OK(`A001`)
		c.C(`A002 LSUB "" "*"`).S(`* LSUB (\Unmarked) "/" "INBOX"`, `* LSUB (\Noselect) "/" "Work"`).OK(`A002`)

		c.C(`A003 UNSUBSCRIBE Work`).OK(`A003`)
		c.C(`A004 LSUB "" "*"`).S(`* LSUB (\Unmarked) "/" "INBOX"`).OK(`A004`)

		require.Empty(t, conn.subscriptions())
	})
}

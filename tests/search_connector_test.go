package tests

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/ProtonMail/gluon/connector"
	"github.com/ProtonMail/gluon/imap"
	"github.com/ProtonMail/gluon/imap/command"
	"github.com/stretchr/testify/require"
)

type searchAnswer func(mboxID imap.MailboxID, keys []command.SearchKey) ([]imap.MessageID, bool, error)

// searchConnector records the searches Gluon hands over and can replace the dummy's answer.
type searchConnector struct {
	*connector.Dummy

	lock   sync.Mutex
	calls  [][]command.SearchKey
	answer searchAnswer
}

func (c *searchConnector) SearchMessages(ctx context.Context, mboxID imap.MailboxID, keys []command.SearchKey) ([]imap.MessageID, bool, error) {
	c.lock.Lock()
	c.calls = append(c.calls, keys)
	answer := c.answer
	c.lock.Unlock()

	if answer != nil {
		return answer(mboxID, keys)
	}

	return c.Dummy.SearchMessages(ctx, mboxID, keys)
}

func (c *searchConnector) setAnswer(answer searchAnswer) {
	c.lock.Lock()
	defer c.lock.Unlock()

	c.answer = answer
}

func (c *searchConnector) searches() [][]command.SearchKey {
	c.lock.Lock()
	defer c.lock.Unlock()

	return c.calls
}

type searchConnectorBuilder struct {
	conn *searchConnector
}

func (b *searchConnectorBuilder) New(usernames []string, password []byte, period time.Duration, flags, permFlags, attrs imap.FlagSet) Connector {
	b.conn = &searchConnector{Dummy: connector.NewDummy(usernames, password, period, flags, permFlags, attrs)}
	b.conn.SetSearchEnabled(true)

	return b.conn
}

type searchFixture struct {
	conn                     *searchConnector
	mboxID                   imap.MailboxID
	invoice, holiday, notice imap.MessageID
}

const (
	searchInvoiceLiteral = "From: billing@pm.me\r\nTo: user@pm.me\r\nDate: Fri, 13 Nov 2026 10:00:00 +0000\r\nSubject: Invoice for March\r\n\r\nPlease pay.\r\n"
	searchHolidayLiteral = "From: friend@pm.me\r\nTo: user@pm.me\r\nDate: Fri, 13 Nov 2026 11:00:00 +0000\r\nSubject: Holiday plans\r\n\r\nSee you in March.\r\n"
	searchNoticeLiteral  = "From: billing@pm.me\r\nTo: user@pm.me\r\nDate: Fri, 13 Nov 2026 12:00:00 +0000\r\nSubject: Invoice for April\r\n\r\nSecond notice.\r\n"
)

func createSearchMessages(c *testConnection, s *testSession) (imap.MailboxID, imap.MessageID, imap.MessageID, imap.MessageID) {
	mboxID := s.mailboxCreated("user", []string{"Archive"})

	invoice := s.messageCreated("user", mboxID, []byte(searchInvoiceLiteral), time.Now(), imap.FlagSeen)
	holiday := s.messageCreated("user", mboxID, []byte(searchHolidayLiteral), time.Now())
	notice := s.messageCreated("user", mboxID, []byte(searchNoticeLiteral), time.Now())

	s.flush("user")

	c.C(`A000 SELECT Archive`).OK(`A000`)

	return mboxID, invoice, holiday, notice
}

func runSearchConnectorTest(t *testing.T, tests func(c *testConnection, f searchFixture)) {
	builder := &searchConnectorBuilder{}

	runOneToOneTestWithAuth(t, defaultServerOptions(t, withConnectorBuilder(builder)), func(c *testConnection, s *testSession) {
		mboxID, invoice, holiday, notice := createSearchMessages(c, s)

		tests(c, searchFixture{conn: builder.conn, mboxID: mboxID, invoice: invoice, holiday: holiday, notice: notice})
	})
}

func TestSearchConnectorAnswersContentKeys(t *testing.T) {
	runSearchConnectorTest(t, func(c *testConnection, f searchFixture) {
		c.C(`A001 SEARCH SUBJECT invoice`).S(`* SEARCH 1 3`).OK(`A001`)
		c.C(`A002 SEARCH TEXT march`).S(`* SEARCH 1 2`).OK(`A002`)
		c.C(`A003 SEARCH OR SUBJECT holiday BODY notice`).S(`* SEARCH 2 3`).OK(`A003`)
		c.C(`A004 SEARCH NOT SUBJECT invoice`).S(`* SEARCH 2`).OK(`A004`)
		c.C(`A005 SEARCH FROM billing TEXT april`).S(`* SEARCH 3`).OK(`A005`)
		c.C(`A006 SEARCH SUBJECT nothing`).S(`* SEARCH`).OK(`A006`)

		require.Equal(t, [][]command.SearchKey{
			{&command.SearchKeySubject{Value: "invoice"}},
			{&command.SearchKeyText{Value: "march"}},
			{&command.SearchKeyOr{Key1: &command.SearchKeySubject{Value: "holiday"}, Key2: &command.SearchKeyBody{Value: "notice"}}},
			{&command.SearchKeyNot{Key: &command.SearchKeySubject{Value: "invoice"}}},
			{&command.SearchKeyFrom{Value: "billing"}, &command.SearchKeyText{Value: "april"}},
			{&command.SearchKeySubject{Value: "nothing"}},
		}, f.conn.searches())
	})
}

func TestSearchConnectorIsAskedForTheRightMailbox(t *testing.T) {
	runSearchConnectorTest(t, func(c *testConnection, f searchFixture) {
		var asked imap.MailboxID

		f.conn.setAnswer(func(mboxID imap.MailboxID, _ []command.SearchKey) ([]imap.MessageID, bool, error) {
			asked = mboxID

			return nil, true, nil
		})

		c.C(`A001 SEARCH SUBJECT invoice`).S(`* SEARCH`).OK(`A001`)

		require.Equal(t, f.mboxID, asked)
	})
}

func TestSearchConnectorLeavesNumbersAndFlagsToGluon(t *testing.T) {
	runSearchConnectorTest(t, func(c *testConnection, f searchFixture) {
		c.C(`A001 UID SEARCH UID 2:* UNSEEN SUBJECT invoice`).S(`* SEARCH 3`).OK(`A001`)
		c.C(`A002 SEARCH 1:2 TEXT march`).S(`* SEARCH 1 2`).OK(`A002`)
		c.C(`A003 SEARCH SEEN SUBJECT invoice`).S(`* SEARCH 1`).OK(`A003`)
		c.C(`A004 SEARCH ALL SUBJECT invoice NOT SEEN`).S(`* SEARCH 3`).OK(`A004`)

		require.Equal(t, [][]command.SearchKey{
			{&command.SearchKeySubject{Value: "invoice"}},
			{&command.SearchKeyText{Value: "march"}},
			{&command.SearchKeySubject{Value: "invoice"}},
			{&command.SearchKeySubject{Value: "invoice"}},
		}, f.conn.searches())
	})
}

func TestSearchConnectorWithESearch(t *testing.T) {
	runSearchConnectorTest(t, func(c *testConnection, f searchFixture) {
		c.C(`A001 SEARCH RETURN (COUNT MIN MAX ALL) SUBJECT invoice`)
		c.S(`* ESEARCH (TAG "A001") COUNT 2 MIN 1 MAX 3 ALL 1,3`)
		c.OK(`A001`)

		c.C(`A002 UID SEARCH RETURN (COUNT) TEXT march`)
		c.S(`* ESEARCH (TAG "A002") UID COUNT 2`)
		c.OK(`A002`)

		c.C(`A003 SEARCH RETURN (COUNT MIN) SUBJECT nothing`)
		c.S(`* ESEARCH (TAG "A003") COUNT 0`)
		c.OK(`A003`)

		require.Len(t, f.conn.searches(), 3)
	})
}

func TestSearchConnectorCustomAnswerIsUsed(t *testing.T) {
	runSearchConnectorTest(t, func(c *testConnection, f searchFixture) {
		f.conn.setAnswer(func(imap.MailboxID, []command.SearchKey) ([]imap.MessageID, bool, error) {
			return []imap.MessageID{f.holiday}, true, nil
		})

		c.C(`A001 SEARCH SUBJECT invoice`).S(`* SEARCH 2`).OK(`A001`)
		c.C(`A002 UID SEARCH UNSEEN SUBJECT invoice`).S(`* SEARCH 2`).OK(`A002`)
		c.C(`A003 SEARCH SEEN SUBJECT invoice`).S(`* SEARCH`).OK(`A003`)
	})
}

func TestSearchConnectorIgnoresMessagesOutsideTheMailbox(t *testing.T) {
	runSearchConnectorTest(t, func(c *testConnection, f searchFixture) {
		f.conn.setAnswer(func(imap.MailboxID, []command.SearchKey) ([]imap.MessageID, bool, error) {
			return []imap.MessageID{"not-a-message", f.notice, f.notice}, true, nil
		})

		c.C(`A001 SEARCH SUBJECT invoice`).S(`* SEARCH 3`).OK(`A001`)
	})
}

func TestSearchConnectorDeclineFallsBackToGluon(t *testing.T) {
	runSearchConnectorTest(t, func(c *testConnection, f searchFixture) {
		c.C(`A001 SEARCH SINCE 1-Jan-2020 SUBJECT invoice`).S(`* SEARCH 1 3`).OK(`A001`)

		f.conn.setAnswer(func(imap.MailboxID, []command.SearchKey) ([]imap.MessageID, bool, error) {
			return []imap.MessageID{f.holiday}, false, nil
		})

		c.C(`A002 SEARCH SUBJECT invoice`).S(`* SEARCH 1 3`).OK(`A002`)
		c.C(`A003 SEARCH RETURN (COUNT) SUBJECT invoice`).S(`* ESEARCH (TAG "A003") COUNT 2`).OK(`A003`)

		require.Len(t, f.conn.searches(), 3)
	})
}

func TestSearchConnectorErrorAnswersNo(t *testing.T) {
	runSearchConnectorTest(t, func(c *testConnection, f searchFixture) {
		f.conn.setAnswer(func(imap.MailboxID, []command.SearchKey) ([]imap.MessageID, bool, error) {
			return nil, false, errors.New("search index is unavailable")
		})

		c.C(`A001 SEARCH SUBJECT invoice`)
		c.Sx(`^A001 NO .*search index is unavailable`)

		c.C(`A002 SEARCH RETURN (COUNT) SUBJECT invoice`)
		c.Sx(`^A002 NO .*search index is unavailable`)

		c.C(`A003 SEARCH UNSEEN`).S(`* SEARCH 2 3`).OK(`A003`)

		f.conn.setAnswer(nil)

		c.C(`A004 SEARCH SUBJECT invoice`).S(`* SEARCH 1 3`).OK(`A004`)
	})
}

func TestSearchConnectorIsNotAskedWhenGluonOwnsTheSearch(t *testing.T) {
	runSearchConnectorTest(t, func(c *testConnection, f searchFixture) {
		c.C(`A001 SEARCH UNSEEN`).S(`* SEARCH 2 3`).OK(`A001`)
		c.C(`A002 SEARCH ALL`).S(`* SEARCH 1 2 3`).OK(`A002`)
		c.C(`A003 SEARCH 1:2`).S(`* SEARCH 1 2`).OK(`A003`)
		c.C(`A004 UID SEARCH UID 3`).S(`* SEARCH 3`).OK(`A004`)
		c.C(`A005 SEARCH OR UNSEEN SUBJECT invoice`).S(`* SEARCH 1 2 3`).OK(`A005`)
		c.C(`A006 SEARCH NOT (SEEN SUBJECT invoice)`).S(`* SEARCH 2 3`).OK(`A006`)
		c.C(`A007 SEARCH LARGER 1`).S(`* SEARCH 1 2 3`).OK(`A007`)

		require.Empty(t, f.conn.searches())
	})
}

func TestSearchConnectorIsNotAskedForOtherCharsets(t *testing.T) {
	runSearchConnectorTest(t, func(c *testConnection, f searchFixture) {
		c.C(`A001 SEARCH CHARSET ISO-8859-1 SUBJECT invoice`).S(`* SEARCH 1 3`).OK(`A001`)

		require.Empty(t, f.conn.searches())

		c.C(`A002 SEARCH CHARSET UTF-8 SUBJECT invoice`).S(`* SEARCH 1 3`).OK(`A002`)
		c.C(`A003 SEARCH CHARSET us-ascii SUBJECT invoice`).S(`* SEARCH 1 3`).OK(`A003`)

		require.Len(t, f.conn.searches(), 2)
	})
}

func TestSearchDummyConnectorAnswersWhenEnabled(t *testing.T) {
	runOneToOneTestWithAuth(t, defaultServerOptions(t), func(c *testConnection, s *testSession) {
		createSearchMessages(c, s)

		c.C(`A001 SEARCH SUBJECT invoice`).S(`* SEARCH 1 3`).OK(`A001`)

		s.conns[s.userIDs["user"]].SetSearchEnabled(true)

		c.C(`A002 SEARCH SUBJECT invoice`).S(`* SEARCH 1 3`).OK(`A002`)
		c.C(`A003 SEARCH RETURN (ALL) UNSEEN TEXT march`).S(`* ESEARCH (TAG "A003") ALL 2`).OK(`A003`)
	})
}

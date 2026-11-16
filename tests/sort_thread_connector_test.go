package tests

import (
	"errors"
	"testing"

	"github.com/ProtonMail/gluon/imap"
	"github.com/ProtonMail/gluon/imap/command"
	"github.com/stretchr/testify/require"
)

func TestSortConnectorAnswersContentKeys(t *testing.T) {
	runSearchConnectorTest(t, func(c *testConnection, f searchFixture) {
		c.C(`A001 SORT (REVERSE DATE) UTF-8 SUBJECT invoice`).S(`* SORT 3 1`).OK(`A001`)
		c.C(`A002 UID SORT (DATE) UTF-8 FROM billing BODY notice`).S(`* SORT 3`).OK(`A002`)
		c.C(`A003 SORT (SUBJECT) UTF-8 TEXT march`).S(`* SORT 2 1`).OK(`A003`)
		c.C(`A004 SORT (DATE) UTF-8 SUBJECT nothing`).S(`* SORT`).OK(`A004`)

		require.Equal(t, [][]command.SearchKey{
			{&command.SearchKeySubject{Value: "invoice"}},
			{&command.SearchKeyFrom{Value: "billing"}, &command.SearchKeyBody{Value: "notice"}},
			{&command.SearchKeyText{Value: "march"}},
			{&command.SearchKeySubject{Value: "nothing"}},
		}, f.conn.searches())
	})
}

func TestSortConnectorLeavesNumbersAndFlagsToGluon(t *testing.T) {
	runSearchConnectorTest(t, func(c *testConnection, f searchFixture) {
		c.C(`A001 SORT (DATE) UTF-8 UNSEEN SUBJECT invoice`).S(`* SORT 3`).OK(`A001`)
		c.C(`A002 SORT (DATE) UTF-8 1:2 SUBJECT invoice`).S(`* SORT 1`).OK(`A002`)

		require.Equal(t, [][]command.SearchKey{
			{&command.SearchKeySubject{Value: "invoice"}},
			{&command.SearchKeySubject{Value: "invoice"}},
		}, f.conn.searches())
	})
}

func TestSortConnectorCustomAnswerIsUsed(t *testing.T) {
	runSearchConnectorTest(t, func(c *testConnection, f searchFixture) {
		f.conn.setAnswer(func(imap.MailboxID, []command.SearchKey) ([]imap.MessageID, bool, error) {
			return []imap.MessageID{f.notice, f.holiday}, true, nil
		})

		c.C(`A001 SORT (DATE) UTF-8 SUBJECT invoice`).S(`* SORT 2 3`).OK(`A001`)
		c.C(`A002 SORT (DATE) UTF-8 SEEN SUBJECT invoice`).S(`* SORT`).OK(`A002`)
	})
}

func TestSortConnectorDeclineFallsBackToGluon(t *testing.T) {
	runSearchConnectorTest(t, func(c *testConnection, f searchFixture) {
		f.conn.setAnswer(func(imap.MailboxID, []command.SearchKey) ([]imap.MessageID, bool, error) {
			return []imap.MessageID{f.holiday}, false, nil
		})

		c.C(`A001 SORT (REVERSE DATE) UTF-8 SUBJECT invoice`).S(`* SORT 3 1`).OK(`A001`)

		require.Len(t, f.conn.searches(), 1)
	})
}

func TestSortConnectorErrorAnswersNo(t *testing.T) {
	runSearchConnectorTest(t, func(c *testConnection, f searchFixture) {
		f.conn.setAnswer(func(imap.MailboxID, []command.SearchKey) ([]imap.MessageID, bool, error) {
			return nil, false, errors.New("search index is unavailable")
		})

		c.C(`A001 SORT (DATE) UTF-8 SUBJECT invoice`)
		c.Sx(`^A001 NO .*search index is unavailable`)

		c.C(`A002 THREAD REFERENCES UTF-8 SUBJECT invoice`)
		c.Sx(`^A002 NO .*search index is unavailable`)

		c.C(`A003 SORT (DATE) UTF-8 UNSEEN`).S(`* SORT 2 3`).OK(`A003`)
	})
}

func TestSortConnectorIsNotAskedWhenGluonOwnsTheSearch(t *testing.T) {
	runSearchConnectorTest(t, func(c *testConnection, f searchFixture) {
		c.C(`A001 SORT (DATE) UTF-8 ALL`).S(`* SORT 1 2 3`).OK(`A001`)
		c.C(`A002 SORT (DATE) UTF-8 OR UNSEEN SUBJECT invoice`).S(`* SORT 1 2 3`).OK(`A002`)
		c.C(`A003 THREAD REFERENCES UTF-8 UNSEEN`).S(`* THREAD (2)(3)`).OK(`A003`)

		require.Empty(t, f.conn.searches())
	})
}

func TestThreadConnectorAnswersContentKeys(t *testing.T) {
	runSearchConnectorTest(t, func(c *testConnection, f searchFixture) {
		c.C(`A001 THREAD REFERENCES UTF-8 SUBJECT invoice`).S(`* THREAD (1)(3)`).OK(`A001`)
		c.C(`A002 UID THREAD ORDEREDSUBJECT UTF-8 UNSEEN SUBJECT invoice`).S(`* THREAD (3)`).OK(`A002`)
		c.C(`A003 THREAD REFERENCES UTF-8 SUBJECT nothing`).S(`* THREAD`).OK(`A003`)

		require.Equal(t, [][]command.SearchKey{
			{&command.SearchKeySubject{Value: "invoice"}},
			{&command.SearchKeySubject{Value: "invoice"}},
			{&command.SearchKeySubject{Value: "nothing"}},
		}, f.conn.searches())
	})
}

func TestThreadConnectorCustomAnswerIsUsed(t *testing.T) {
	runSearchConnectorTest(t, func(c *testConnection, f searchFixture) {
		f.conn.setAnswer(func(imap.MailboxID, []command.SearchKey) ([]imap.MessageID, bool, error) {
			return []imap.MessageID{f.holiday}, true, nil
		})

		c.C(`A001 THREAD REFERENCES UTF-8 SUBJECT invoice`).S(`* THREAD (2)`).OK(`A001`)
	})
}

func TestSearchConnectorIsGivenUTF8ForOtherCharsets(t *testing.T) {
	runSearchConnectorTest(t, func(c *testConnection, f searchFixture) {
		c.C(`A001 SEARCH CHARSET ISO-8859-1 SUBJECT invoice`).S(`* SEARCH 1 3`).OK(`A001`)
		c.Cb([]byte("A002 SEARCH CHARSET ISO-8859-1 OR SUBJECT {4+}\r\ncaf\xe9 NOT BODY {5+}\r\nna\xefve")).S(`* SEARCH 1 2 3`).OK(`A002`)
		c.Cb([]byte("A003 SORT (DATE) ISO-8859-1 HEADER X-Place {6+}\r\nM\xfcnich")).S(`* SORT`).OK(`A003`)

		require.Equal(t, [][]command.SearchKey{
			{&command.SearchKeySubject{Value: "invoice"}},
			{&command.SearchKeyOr{
				Key1: &command.SearchKeySubject{Value: "café"},
				Key2: &command.SearchKeyNot{Key: &command.SearchKeyBody{Value: "naïve"}},
			}},
			{&command.SearchKeyHeader{Field: "X-Place", Value: "Münich"}},
		}, f.conn.searches())
	})
}

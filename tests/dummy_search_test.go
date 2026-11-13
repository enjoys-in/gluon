package tests

import (
	"context"
	"slices"
	"testing"
	"time"

	"github.com/ProtonMail/gluon/connector"
	"github.com/ProtonMail/gluon/internal/utils"

	"github.com/ProtonMail/gluon/imap"
	"github.com/ProtonMail/gluon/imap/command"
	"github.com/stretchr/testify/require"
)

const (
	searchTestInvoice = "From: billing@example.com\r\nTo: user@example.com\r\nSubject: Invoice for March\r\n\r\nPlease pay the invoice.\r\n"
	searchTestHoliday = "From: friend@example.com\r\nTo: user@example.com\r\nCc: other@example.com\r\nSubject: Holiday plans\r\n\r\nLet us meet in March.\r\n"
)

func newSearchTestDummy(t *testing.T) (*connector.Dummy, imap.MailboxID, imap.MessageID, imap.MessageID) {
	t.Helper()

	conn := connector.NewDummy([]string{"username"}, []byte("password"), defaultPeriod, defaultFlags, defaultPermanentFlags, defaultAttributes)
	conn.SetSearchEnabled(true)

	mboxID := newDummyMailbox(t, conn, "Archive")

	create := func(literal string) imap.MessageID {
		messageID := imap.MessageID(utils.NewRandomMessageID())

		require.NoError(t, conn.MessageCreated(
			imap.Message{ID: messageID, Flags: imap.NewFlagSet(), Date: time.Now()},
			[]byte(literal),
			[]imap.MailboxID{mboxID},
		))

		return messageID
	}

	invoice, holiday := create(searchTestInvoice), create(searchTestHoliday)

	// No server reads the updates of this connector.
	conn.ClearUpdates()

	return conn, mboxID, invoice, holiday
}

func newDummyMailbox(t *testing.T, conn *connector.Dummy, name string) imap.MailboxID {
	t.Helper()

	mboxID := imap.MailboxID(utils.NewRandomMailboxID())

	require.NoError(t, conn.MailboxCreated(imap.Mailbox{ID: mboxID, Name: []string{name}}))

	return mboxID
}

func searchDummy(t *testing.T, conn *connector.Dummy, mboxID imap.MailboxID, keys ...command.SearchKey) []imap.MessageID {
	t.Helper()

	messageIDs, ok, err := conn.SearchMessages(context.Background(), mboxID, keys)
	require.NoError(t, err)
	require.True(t, ok)

	slices.Sort(messageIDs)

	return messageIDs
}

func sortedIDs(messageIDs ...imap.MessageID) []imap.MessageID {
	slices.Sort(messageIDs)

	return messageIDs
}

func TestDummyConnector_SearchMessages(t *testing.T) {
	conn, mboxID, invoice, holiday := newSearchTestDummy(t)

	tests := []struct {
		name string
		keys []command.SearchKey
		want []imap.MessageID
	}{
		{name: "subject", keys: []command.SearchKey{&command.SearchKeySubject{Value: "invoice"}}, want: sortedIDs(invoice)},
		{name: "from", keys: []command.SearchKey{&command.SearchKeyFrom{Value: "FRIEND@"}}, want: sortedIDs(holiday)},
		{name: "to", keys: []command.SearchKey{&command.SearchKeyTo{Value: "user@example.com"}}, want: sortedIDs(invoice, holiday)},
		{name: "cc", keys: []command.SearchKey{&command.SearchKeyCC{Value: "other"}}, want: sortedIDs(holiday)},
		{name: "bcc", keys: []command.SearchKey{&command.SearchKeyBCC{Value: "other"}}, want: nil},
		{name: "header", keys: []command.SearchKey{&command.SearchKeyHeader{Field: "Subject", Value: "plans"}}, want: sortedIDs(holiday)},
		{name: "body", keys: []command.SearchKey{&command.SearchKeyBody{Value: "march"}}, want: sortedIDs(holiday)},
		{name: "text", keys: []command.SearchKey{&command.SearchKeyText{Value: "march"}}, want: sortedIDs(invoice, holiday)},
		{name: "not", keys: []command.SearchKey{&command.SearchKeyNot{Key: &command.SearchKeySubject{Value: "invoice"}}}, want: sortedIDs(holiday)},
		{
			name: "or",
			keys: []command.SearchKey{&command.SearchKeyOr{
				Key1: &command.SearchKeySubject{Value: "invoice"},
				Key2: &command.SearchKeySubject{Value: "holiday"},
			}},
			want: sortedIDs(invoice, holiday),
		},
		{
			name: "every key has to match",
			keys: []command.SearchKey{&command.SearchKeyText{Value: "march"}, &command.SearchKeyFrom{Value: "billing"}},
			want: sortedIDs(invoice),
		},
		{name: "no match", keys: []command.SearchKey{&command.SearchKeySubject{Value: "nothing"}}, want: nil},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.want, searchDummy(t, conn, mboxID, tc.keys...))
		})
	}
}

func TestDummyConnector_SearchMessagesOnlyLooksInTheMailbox(t *testing.T) {
	conn, _, _, _ := newSearchTestDummy(t)

	other := newDummyMailbox(t, conn, "Other")

	require.Empty(t, searchDummy(t, conn, other, &command.SearchKeyText{Value: "march"}))
}

func TestDummyConnector_SearchMessagesDeclinesWhatItCannotEvaluate(t *testing.T) {
	conn, mboxID, _, _ := newSearchTestDummy(t)

	for name, keys := range map[string][]command.SearchKey{
		"date key":           {&command.SearchKeySince{Value: time.Now()}},
		"flag key":           {&command.SearchKeySeen{}},
		"unsupported in not": {&command.SearchKeyNot{Key: &command.SearchKeySeen{}}},
		"unsupported in or":  {&command.SearchKeyOr{Key1: &command.SearchKeySubject{Value: "invoice"}, Key2: &command.SearchKeySeen{}}},
		"unsupported second": {&command.SearchKeySubject{Value: "invoice"}, &command.SearchKeySince{Value: time.Now()}},
	} {
		t.Run(name, func(t *testing.T) {
			messageIDs, ok, err := conn.SearchMessages(context.Background(), mboxID, keys)
			require.NoError(t, err)
			require.False(t, ok)
			require.Empty(t, messageIDs)
		})
	}
}

func TestDummyConnector_SearchMessagesIsOffByDefault(t *testing.T) {
	conn, mboxID, _, _ := newSearchTestDummy(t)
	conn.SetSearchEnabled(false)

	messageIDs, ok, err := conn.SearchMessages(context.Background(), mboxID, []command.SearchKey{&command.SearchKeyText{Value: "march"}})
	require.NoError(t, err)
	require.False(t, ok)
	require.Empty(t, messageIDs)
}

func TestDummyConnector_SearchMessagesUnknownMailbox(t *testing.T) {
	conn, _, _, _ := newSearchTestDummy(t)

	_, ok, err := conn.SearchMessages(context.Background(), "no-such-mailbox", []command.SearchKey{&command.SearchKeyText{Value: "march"}})
	require.ErrorIs(t, err, connector.ErrNoSuchMailbox)
	require.False(t, ok)
}

package tests

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/ProtonMail/gluon/connector"
	"github.com/ProtonMail/gluon/imap"
)

// multiAppend builds an APPEND command with one non-synchronizing literal per message. A message is its literal,
// optionally preceded by flags and a date: "(\Seen) |literal".
func multiAppend(tag, mailbox string, messages ...string) []byte {
	command := fmt.Sprintf("%v APPEND %v", tag, mailbox)

	for _, message := range messages {
		prefix, literal, hasPrefix := strings.Cut(message, "|")
		if !hasPrefix {
			prefix, literal = "", message
		}

		command += fmt.Sprintf(" %v{%v+}\r\n%v", prefix, len(literal), literal)
	}

	return []byte(command)
}

func TestMultiAppend(t *testing.T) {
	runOneToOneTestWithAuth(t, defaultServerOptions(t), func(c *testConnection, _ *testSession) {
		c.Cb(multiAppend("A001", "INBOX",
			`(\Seen) |`+buildRFC5322TestLiteral("To: 1@pm.me"),
			`(\Flagged) "13-Nov-2026 10:00:00 +0000" |`+buildRFC5322TestLiteral("To: 2@pm.me"),
			buildRFC5322TestLiteral("To: 3@pm.me"),
		))
		c.Sx(`^A001 OK \[APPENDUID \d+ 1:3\] APPEND`)

		c.C(`A002 SELECT INBOX`).OK(`A002`)
		c.C(`A003 SEARCH SEEN`).S(`* SEARCH 1`).OK(`A003`)
		c.C(`A004 SEARCH FLAGGED`).S(`* SEARCH 2`).OK(`A004`)
		c.C(`A005 SEARCH TO 3@pm.me`).S(`* SEARCH 3`).OK(`A005`)
		c.C(`A006 FETCH 2 (INTERNALDATE)`).S(`* 2 FETCH (INTERNALDATE "13-Nov-2026 10:00:00 +0000")`).OK(`A006`)
	})
}

func TestMultiAppendWithSynchronizingLiterals(t *testing.T) {
	runOneToOneTestWithAuth(t, defaultServerOptions(t), func(c *testConnection, _ *testSession) {
		first := buildRFC5322TestLiteral("To: 1@pm.me")
		second := buildRFC5322TestLiteral("To: 2@pm.me")

		c.Cf(`A001 APPEND INBOX {%v}`, len(first))
		c.Sx(`\+.*`)
		c.Cf(`%v (\Seen) {%v}`, first, len(second))
		c.Sx(`\+.*`)
		c.C(second)
		c.Sx(`^A001 OK \[APPENDUID \d+ 1:2\] APPEND`)

		c.C(`A002 STATUS INBOX (MESSAGES UNSEEN)`).S(`* STATUS "INBOX" (MESSAGES 2 UNSEEN 1)`).OK(`A002`)
	})
}

func TestMultiAppendToTheSelectedMailbox(t *testing.T) {
	runOneToOneTestWithAuth(t, defaultServerOptions(t), func(c *testConnection, _ *testSession) {
		c.C(`A001 SELECT INBOX`).OK(`A001`)

		c.Cb(multiAppend("A002", "INBOX", buildRFC5322TestLiteral("To: 1@pm.me"), buildRFC5322TestLiteral("To: 2@pm.me")))
		c.S(`* 2 EXISTS`, `* 2 RECENT`)
		c.Sx(`^A002 OK \[APPENDUID \d+ 1:2\] APPEND`)

		c.Cb(multiAppend("A003", "INBOX", buildRFC5322TestLiteral("To: 3@pm.me"), buildRFC5322TestLiteral("To: 4@pm.me")))
		c.S(`* 4 EXISTS`, `* 4 RECENT`)
		c.Sx(`^A003 OK \[APPENDUID \d+ 3:4\] APPEND`)
	})
}

func TestMultiAppendChecksEveryMessageFirst(t *testing.T) {
	runOneToOneTestWithAuth(t, defaultServerOptions(t), func(c *testConnection, _ *testSession) {
		// The second message has no From header.
		c.Cb(multiAppend("A001", "INBOX", buildRFC5322TestLiteral("To: 1@pm.me"), "Subject: incomplete\r\n\r\nbody"))
		c.Sx(`^A001 BAD .*invalid rfc5322 message`)

		c.Cb(multiAppend("A002", "INBOX", buildRFC5322TestLiteral("To: 1@pm.me"), `(\Recent) |`+buildRFC5322TestLiteral("To: 2@pm.me")))
		c.BAD(`A002`)

		c.Cb(multiAppend("A003", "Missing", buildRFC5322TestLiteral("To: 1@pm.me"), buildRFC5322TestLiteral("To: 2@pm.me")))
		c.Sx(`^A003 NO \[TRYCREATE\]`)

		c.C(`A004 STATUS INBOX (MESSAGES UIDNEXT)`).S(`* STATUS "INBOX" (MESSAGES 0 UIDNEXT 1)`).OK(`A004`)

		c.C(`A005 CAPABILITY`)
		c.Sx(`\* CAPABILITY .* MULTIAPPEND `)
		c.OK(`A005`)
	})
}

func TestMultiAppendNeedsLogin(t *testing.T) {
	runOneToOneTest(t, defaultServerOptions(t), func(c *testConnection, _ *testSession) {
		c.Cb(multiAppend("A001", "INBOX", buildRFC5322TestLiteral("To: 1@pm.me"), buildRFC5322TestLiteral("To: 2@pm.me")))
		c.NO(`A001`)
		c.C(`A002 NOOP`).OK(`A002`)
	})
}

// refusingConnector refuses the messages that say so. As the connector contract asks, it creates none of the
// messages of a command when it refuses one of them.
type refusingConnector struct {
	*connector.Dummy
}

func (c *refusingConnector) CreateMessages(ctx context.Context, cache connector.IMAPStateWrite, mboxID imap.MailboxID, messages []connector.MessageToCreate) ([]imap.Message, [][]byte, error) {
	for _, message := range messages {
		if bytes.Contains(message.Literal, []byte("refuse this message")) {
			return nil, nil, errors.New("the storage is full")
		}
	}

	return c.Dummy.CreateMessages(ctx, cache, mboxID, messages)
}

type refusingConnectorBuilder struct{}

func (refusingConnectorBuilder) New(usernames []string, password []byte, period time.Duration, flags, permFlags, attrs imap.FlagSet) Connector {
	return &refusingConnector{Dummy: connector.NewDummy(usernames, password, period, flags, permFlags, attrs)}
}

func TestMultiAppendAppendsNothingWhenTheConnectorRefusesOneMessage(t *testing.T) {
	runManyToOneTestWithAuth(t, defaultServerOptions(t, withConnectorBuilder(refusingConnectorBuilder{})), []int{1, 2}, func(c map[int]*testConnection, _ *testSession) {
		c[1].C(`A001 CREATE Work`).OK(`A001`)
		c[1].doAppend(`Work`, buildRFC5322TestLiteral("To: 0@pm.me")).expect("OK")
		c[2].C(`B001 SELECT Work`).OK(`B001`)

		c[1].Cb(multiAppend("A002", "Work",
			buildRFC5322TestLiteral("To: 1@pm.me"),
			buildRFC5322TestLiteral("To: 2@pm.me"),
			buildRFC5322TestLiteral("Subject: refuse this message"),
			buildRFC5322TestLiteral("To: 4@pm.me"),
		))
		c[1].Sx(`^A002 NO .*the storage is full`)

		c[1].C(`A003 STATUS Work (MESSAGES UIDNEXT)`).S(`* STATUS "Work" (MESSAGES 1 UIDNEXT 2)`).OK(`A003`)

		// The session that has the mailbox selected never sees any of the messages.
		c[2].C(`B002 NOOP`).Sx(`^B002 OK `)
		c[2].C(`B003 UID SEARCH ALL`).S(`* SEARCH 1`).OK(`B003`)

		// Like a single APPEND that fails, the messages are kept in the recovery mailbox.
		c[1].C(`A004 STATUS "Recovered Messages" (MESSAGES)`).S(`* STATUS "Recovered Messages" (MESSAGES 4)`).OK(`A004`)

		// A later command works and its UIDs follow the last message that was appended.
		c[1].Cb(multiAppend("A005", "Work", buildRFC5322TestLiteral("To: 5@pm.me"), buildRFC5322TestLiteral("To: 6@pm.me")))
		c[1].Sx(`^A005 OK \[APPENDUID \d+ 2:3\] APPEND`)

		c[2].C(`B004 NOOP`)
		c[2].S(`* 3 EXISTS`, `* 3 RECENT`)
		c[2].OK(`B004`)
		c[2].C(`B005 UID SEARCH ALL`).S(`* SEARCH 1 2 3`).OK(`B005`)
	})
}

func TestMultiAppendSurvivesTheConnectorUpdates(t *testing.T) {
	runOneToOneTestWithAuth(t, defaultServerOptions(t), func(c *testConnection, s *testSession) {
		c.Cb(multiAppend("A001", "INBOX", buildRFC5322TestLiteral("To: 1@pm.me"), buildRFC5322TestLiteral("To: 2@pm.me")))
		c.Sx(`^A001 OK \[APPENDUID \d+ 1:2\] APPEND`)

		// The connector reports the messages it created. They are the ones Gluon already has.
		s.flush("user")

		c.C(`A002 STATUS INBOX (MESSAGES UIDNEXT)`).S(`* STATUS "INBOX" (MESSAGES 2 UIDNEXT 3)`).OK(`A002`)
	})
}

func TestMultiAppendToDrafts(t *testing.T) {
	runOneToOneTestWithAuth(t, defaultServerOptions(t), func(c *testConnection, s *testSession) {
		s.mailboxCreatedWithAttributes("user", []string{"Drafts"}, imap.NewFlagSet(imap.AttrDrafts))
		s.flush("user")

		// A draft does not need the headers of a complete message.
		c.Cb(multiAppend("A001", "Drafts", "Subject: first draft\r\n\r\nbody", "Subject: second draft\r\n\r\nbody"))
		c.Sx(`^A001 OK \[APPENDUID \d+ 1:2\] APPEND`)

		c.C(`A002 STATUS Drafts (MESSAGES)`).S(`* STATUS "Drafts" (MESSAGES 2)`).OK(`A002`)
	})
}

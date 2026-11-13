package tests

import (
	"fmt"
	"testing"
	"time"
)

// Gluon stores each message with its internal ID header: "X-Pm-Gluon-Id: <uuid>\r\n".
const internalIDHeaderSize = 53

func storedSize(literals ...string) int {
	size := 0

	for _, literal := range literals {
		size += len(literal) + internalIDHeaderSize
	}

	return size
}

func TestStatusSize(t *testing.T) {
	runOneToOneTestWithAuth(t, defaultServerOptions(t), func(c *testConnection, _ *testSession) {
		first := buildRFC5322TestLiteral(`To: 1@pm.me`)
		second := buildRFC5322TestLiteral(`To: 2@pm.me`) + "a longer body"

		c.C(`A001 CREATE Work`).OK(`A001`)
		c.doAppend(`Work`, first).expect("OK")
		c.doAppend(`Work`, second, `\Seen`).expect("OK")

		c.C(`A002 STATUS Work (SIZE)`)
		c.S(fmt.Sprintf(`* STATUS "Work" (SIZE %v)`, storedSize(first, second)))
		c.OK(`A002`)

		c.C(`A003 STATUS Work (MESSAGES SIZE UNSEEN)`)
		c.S(fmt.Sprintf(`* STATUS "Work" (MESSAGES 2 SIZE %v UNSEEN 1)`, storedSize(first, second)))
		c.OK(`A003`)

		c.C(`A004 STATUS INBOX (SIZE)`)
		c.S(`* STATUS "INBOX" (SIZE 0)`)
		c.OK(`A004`)
	})
}

func TestStatusSizeMatchesTheMessageSizes(t *testing.T) {
	runOneToOneTestWithAuth(t, defaultServerOptions(t), func(c *testConnection, _ *testSession) {
		literal := buildRFC5322TestLiteral(`To: 1@pm.me`)

		c.doAppend(`INBOX`, literal).expect("OK")
		c.C(`A001 SELECT INBOX`).OK(`A001`)

		c.C(`A002 FETCH 1 (RFC822.SIZE)`)
		c.S(fmt.Sprintf(`* 1 FETCH (RFC822.SIZE %v)`, storedSize(literal)))
		c.OK(`A002`)

		// The selected mailbox reports its size too.
		c.C(`A003 STATUS INBOX (MESSAGES SIZE)`)
		c.S(fmt.Sprintf(`* STATUS "INBOX" (MESSAGES 1 SIZE %v)`, storedSize(literal)))
		c.OK(`A003`)
	})
}

func TestStatusSizeAfterExpunge(t *testing.T) {
	runOneToOneTestWithAuth(t, defaultServerOptions(t), func(c *testConnection, _ *testSession) {
		first := buildRFC5322TestLiteral(`To: 1@pm.me`)
		second := buildRFC5322TestLiteral(`To: 2@pm.me`) + "a longer body"

		c.C(`A001 CREATE Work`).OK(`A001`)
		c.doAppend(`Work`, first).expect("OK")
		c.doAppend(`Work`, second).expect("OK")

		c.C(`A002 SELECT Work`).OK(`A002`)
		c.C(`A003 STORE 1 +FLAGS.SILENT (\Deleted)`).OK(`A003`)
		c.C(`A004 EXPUNGE`).OK(`A004`)
		c.C(`A005 UNSELECT`).OK(`A005`)

		c.C(`A006 STATUS Work (MESSAGES SIZE)`)
		c.S(fmt.Sprintf(`* STATUS "Work" (MESSAGES 1 SIZE %v)`, storedSize(second)))
		c.OK(`A006`)
	})
}

func TestStatusSizeCountsMessagesFromTheConnector(t *testing.T) {
	runOneToOneTestWithAuth(t, defaultServerOptions(t), func(c *testConnection, s *testSession) {
		literal := buildRFC5322TestLiteral(`To: 1@pm.me`)

		mboxID := s.mailboxCreated("user", []string{"Archive"})
		s.messageCreated("user", mboxID, []byte(literal), time.Now())
		s.messageCreated("user", mboxID, []byte(literal), time.Now())
		s.flush("user")

		c.C(`A001 STATUS Archive (SIZE)`)
		c.S(fmt.Sprintf(`* STATUS "Archive" (SIZE %v)`, storedSize(literal, literal)))
		c.OK(`A001`)
	})
}

func TestStatusSizeInListStatus(t *testing.T) {
	runOneToOneTestWithAuth(t, defaultServerOptions(t), func(c *testConnection, _ *testSession) {
		literal := buildRFC5322TestLiteral(`To: 1@pm.me`)

		c.doAppend(`INBOX`, literal).expect("OK")

		c.C(`A001 LIST "" "*" RETURN (STATUS (MESSAGES SIZE))`)
		c.S(
			`* LIST (\HasNoChildren \Marked) "/" "INBOX"`,
			fmt.Sprintf(`* STATUS "INBOX" (MESSAGES 1 SIZE %v)`, storedSize(literal)),
		)
		c.OK(`A001`)
	})
}

func TestStatusSizeFailures(t *testing.T) {
	runOneToOneTest(t, defaultServerOptions(t), func(c *testConnection, _ *testSession) {
		c.C(`A001 STATUS INBOX (SIZE)`).NO(`A001`)

		c.C(`A002 LOGIN user pass`).OK(`A002`)

		c.C(`A003 STATUS Missing (SIZE)`).NO(`A003`)
		c.C(`A004 STATUS INBOX (SIZES)`).BAD(`A004`)
		c.C(`A005 STATUS INBOX (SIZE`).BAD(`A005`)

		c.C(`A006 CAPABILITY`)
		c.Sx(`\* CAPABILITY .* STATUS=SIZE `)
		c.OK(`A006`)
	})
}

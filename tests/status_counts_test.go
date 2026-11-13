package tests

import (
	"testing"
)

const allStatusItems = `(MESSAGES RECENT UIDNEXT UIDVALIDITY UNSEEN)`

func TestStatusOfSelectedAndUnselectedMailbox(t *testing.T) {
	runOneToOneTestWithAuth(t, defaultServerOptions(t), func(c *testConnection, _ *testSession) {
		c.C(`A001 CREATE Work`).OK(`A001`)

		c.doAppend(`Work`, buildRFC5322TestLiteral(`To: 1@pm.me`), `\Seen`).expect("OK")
		c.doAppend(`Work`, buildRFC5322TestLiteral(`To: 2@pm.me`)).expect("OK")
		c.doAppend(`Work`, buildRFC5322TestLiteral(`To: 3@pm.me`), `\Flagged`).expect("OK")

		c.C(`A002 SELECT Work`).OK(`A002`)
		c.C(`A003 STORE 2 +FLAGS.SILENT (\Deleted)`).OK(`A003`)

		// The session that selected the mailbox still sees its messages as recent.
		c.Cf(`A004 STATUS Work %v`, allStatusItems)
		c.Sx(`^\* STATUS "Work" \(MESSAGES 3 RECENT 3 UIDNEXT 4 UIDVALIDITY \d+ UNSEEN 2\)`)
		c.OK(`A004`)

		c.C(`A005 SELECT INBOX`).OK(`A005`)

		c.Cf(`A006 STATUS Work %v`, allStatusItems)
		c.Sx(`^\* STATUS "Work" \(MESSAGES 3 RECENT 0 UIDNEXT 4 UIDVALIDITY \d+ UNSEEN 2\)`)
		c.OK(`A006`)

		c.C(`A007 UNSELECT`).OK(`A007`)

		c.Cf(`A008 STATUS Work %v`, allStatusItems)
		c.Sx(`^\* STATUS "Work" \(MESSAGES 3 RECENT 0 UIDNEXT 4 UIDVALIDITY \d+ UNSEEN 2\)`)
		c.OK(`A008`)
	})
}

func TestStatusCountsRecentMessagesInUnselectedMailbox(t *testing.T) {
	runOneToOneTestWithAuth(t, defaultServerOptions(t), func(c *testConnection, _ *testSession) {
		c.C(`A001 CREATE Work`).OK(`A001`)

		c.doAppend(`Work`, buildRFC5322TestLiteral(`To: 1@pm.me`)).expect("OK")
		c.doAppend(`Work`, buildRFC5322TestLiteral(`To: 2@pm.me`), `\Seen`).expect("OK")

		c.C(`A002 STATUS Work (MESSAGES RECENT UNSEEN)`)
		c.S(`* STATUS "Work" (MESSAGES 2 RECENT 2 UNSEEN 1)`)
		c.OK(`A002`)

		c.C(`A003 STATUS INBOX (MESSAGES RECENT UNSEEN UIDNEXT)`)
		c.S(`* STATUS "INBOX" (MESSAGES 0 RECENT 0 UNSEEN 0 UIDNEXT 1)`)
		c.OK(`A003`)
	})
}

func TestStatusUnseenIgnoresTheCaseOfTheSeenFlag(t *testing.T) {
	runOneToOneTestWithAuth(t, defaultServerOptions(t), func(c *testConnection, _ *testSession) {
		c.C(`A001 CREATE Work`).OK(`A001`)

		c.doAppend(`Work`, buildRFC5322TestLiteral(`To: 1@pm.me`)).expect("OK")
		c.doAppend(`Work`, buildRFC5322TestLiteral(`To: 2@pm.me`)).expect("OK")
		c.doAppend(`Work`, buildRFC5322TestLiteral(`To: 3@pm.me`)).expect("OK")
		c.doAppend(`Work`, buildRFC5322TestLiteral(`To: 4@pm.me`)).expect("OK")

		c.C(`A002 SELECT Work`).OK(`A002`)
		c.C(`A003 STORE 1 +FLAGS.SILENT (\seen)`).OK(`A003`)
		c.C(`A004 STORE 2 +FLAGS.SILENT (\SEEN)`).OK(`A004`)
		c.C(`A005 STORE 3 +FLAGS.SILENT (\Seen)`).OK(`A005`)
		c.C(`A006 UNSELECT`).OK(`A006`)

		c.C(`A007 STATUS Work (MESSAGES UNSEEN)`)
		c.S(`* STATUS "Work" (MESSAGES 4 UNSEEN 1)`)
		c.OK(`A007`)

		c.C(`A008 LIST "" Work RETURN (STATUS (UNSEEN))`)
		c.S(`* LIST (\HasNoChildren \Unmarked) "/" "Work"`)
		c.S(`* STATUS "Work" (UNSEEN 1)`)
		c.S(`A008 OK LIST`)
	})
}

func TestStatusOfMissingMailbox(t *testing.T) {
	runOneToOneTestWithAuth(t, defaultServerOptions(t), func(c *testConnection, _ *testSession) {
		c.C(`A001 STATUS Missing (MESSAGES UNSEEN)`)
		c.Sx(`^A001 NO`)

		c.C(`A002 STATUS INBOX (MESSAGES)`)
		c.S(`* STATUS "INBOX" (MESSAGES 0)`)
		c.OK(`A002`)
	})
}

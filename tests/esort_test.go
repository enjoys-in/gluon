package tests

import (
	"testing"
)

// createSortMessages appends three messages whose date order is 2, 3, 1.
func createSortMessages(c *testConnection) {
	c.doAppend("INBOX", "Date: Mon, 01 Jan 2024 10:00:00 +0000\r\nFrom: c@test.com\r\nSubject: Third\r\n\r\nbody3").expect("OK")
	c.doAppend("INBOX", "Date: Mon, 01 Jan 2023 10:00:00 +0000\r\nFrom: a@test.com\r\nSubject: First\r\n\r\nbody1", `\Seen`).expect("OK")
	c.doAppend("INBOX", "Date: Mon, 01 Jul 2023 10:00:00 +0000\r\nFrom: b@test.com\r\nSubject: Second\r\n\r\nbody2").expect("OK")

	c.C(`A000 SELECT INBOX`).OK(`A000`)
}

func TestESortAll(t *testing.T) {
	runOneToOneTestWithAuth(t, defaultServerOptions(t), func(c *testConnection, _ *testSession) {
		createSortMessages(c)

		c.C(`A001 SORT RETURN (ALL) (DATE) UTF-8 ALL`).S(`* ESEARCH (TAG "A001") ALL 2,3,1`).OK(`A001`)
		c.C(`A002 SORT RETURN () (REVERSE DATE) UTF-8 ALL`).S(`* ESEARCH (TAG "A002") ALL 1,3,2`).OK(`A002`)
		c.C(`A003 SORT RETURN (ALL) (SUBJECT) UTF-8 UNSEEN`).S(`* ESEARCH (TAG "A003") ALL 3,1`).OK(`A003`)
	})
}

func TestESortMinMaxCount(t *testing.T) {
	runOneToOneTestWithAuth(t, defaultServerOptions(t), func(c *testConnection, _ *testSession) {
		createSortMessages(c)

		// MIN and MAX are the first and the last message in sort order.
		c.C(`A001 SORT RETURN (MIN MAX COUNT) (DATE) UTF-8 ALL`).S(`* ESEARCH (TAG "A001") MIN 2 MAX 1 COUNT 3`).OK(`A001`)
		c.C(`A002 SORT RETURN (COUNT) (DATE) UTF-8 SEEN`).S(`* ESEARCH (TAG "A002") COUNT 1`).OK(`A002`)
		c.C(`A003 SORT RETURN (MAX) (REVERSE DATE) UTF-8 ALL`).S(`* ESEARCH (TAG "A003") MAX 2`).OK(`A003`)
	})
}

func TestESortUID(t *testing.T) {
	runOneToOneTestWithAuth(t, defaultServerOptions(t), func(c *testConnection, _ *testSession) {
		createSortMessages(c)

		c.C(`A001 STORE 1 +FLAGS.SILENT (\Deleted)`).OK(`A001`)
		c.C(`A002 EXPUNGE`).OK(`A002`)

		c.C(`A003 UID SORT RETURN (ALL COUNT) (REVERSE DATE) UTF-8 ALL`).S(`* ESEARCH (TAG "A003") UID ALL 3,2 COUNT 2`).OK(`A003`)
		c.C(`A004 SORT RETURN (ALL) (REVERSE DATE) UTF-8 ALL`).S(`* ESEARCH (TAG "A004") ALL 2,1`).OK(`A004`)
	})
}

func TestESortNoMatch(t *testing.T) {
	runOneToOneTestWithAuth(t, defaultServerOptions(t), func(c *testConnection, _ *testSession) {
		createSortMessages(c)

		c.C(`A001 SORT RETURN (MIN MAX ALL COUNT) (DATE) UTF-8 SUBJECT nothing`).S(`* ESEARCH (TAG "A001") COUNT 0`).OK(`A001`)
		c.C(`A002 SORT RETURN (ALL) (DATE) UTF-8 SUBJECT nothing`).S(`* ESEARCH (TAG "A002")`).OK(`A002`)
	})
}

func TestESortLeavesPlainSortUnchanged(t *testing.T) {
	runOneToOneTestWithAuth(t, defaultServerOptions(t), func(c *testConnection, _ *testSession) {
		createSortMessages(c)

		c.C(`A001 SORT (DATE) UTF-8 ALL`).S(`* SORT 2 3 1`).OK(`A001`)
		c.C(`A002 UID SORT (REVERSE DATE) UTF-8 ALL`).S(`* SORT 1 3 2`).OK(`A002`)
	})
}

func TestESortFailures(t *testing.T) {
	runOneToOneTestWithAuth(t, defaultServerOptions(t), func(c *testConnection, _ *testSession) {
		createSortMessages(c)

		c.C(`A001 SORT RETURN (SAVE) (DATE) UTF-8 ALL`).BAD(`A001`)
		c.C(`A002 SORT RETURN (DATE) UTF-8 ALL`).BAD(`A002`)
		c.C(`A003 SORT RETURN (ALL)`).BAD(`A003`)
		c.C(`A004 SORT RETURNS (ALL) (DATE) UTF-8 ALL`).BAD(`A004`)
		c.C(`A005 SORT RETURN (ALL) (DATE) UTF-8`).BAD(`A005`)
		c.C(`A006 SORT RETURN (ALL) (DATE) NO-SUCH-CHARSET ALL`).NO(`A006`)

		c.C(`A007 SORT RETURN (COUNT) (DATE) UTF-8 ALL`).S(`* ESEARCH (TAG "A007") COUNT 3`).OK(`A007`)

		c.C(`A008 CAPABILITY`)
		c.Sx(`\* CAPABILITY .* ESORT `)
		c.OK(`A008`)
	})
}

func TestESortNeedsASelectedMailbox(t *testing.T) {
	runOneToOneTestWithAuth(t, defaultServerOptions(t), func(c *testConnection, _ *testSession) {
		c.C(`A001 SORT RETURN (ALL) (DATE) UTF-8 ALL`).NO(`A001`)
	})
}

func TestESortWithConnectorSearch(t *testing.T) {
	runSearchConnectorTest(t, func(c *testConnection, f searchFixture) {
		c.C(`A001 SORT RETURN (ALL COUNT) (REVERSE DATE) UTF-8 SUBJECT invoice`).S(`* ESEARCH (TAG "A001") ALL 3,1 COUNT 2`).OK(`A001`)
	})
}

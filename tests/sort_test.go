package tests

import (
	"testing"
)

func TestSortByDate(t *testing.T) {
	runOneToOneTestWithAuth(t, defaultServerOptions(t), func(c *testConnection, s *testSession) {
		c.C(`tag select inbox`).OK("tag")

		// Append messages with different dates.
		c.doAppend("inbox", "Date: Mon, 01 Jan 2024 10:00:00 +0000\r\nFrom: c@test.com\r\nSubject: Third\r\n\r\nbody3").expect("OK")
		c.doAppend("inbox", "Date: Mon, 01 Jan 2023 10:00:00 +0000\r\nFrom: a@test.com\r\nSubject: First\r\n\r\nbody1").expect("OK")
		c.doAppend("inbox", "Date: Mon, 01 Jul 2023 10:00:00 +0000\r\nFrom: b@test.com\r\nSubject: Second\r\n\r\nbody2").expect("OK")

		// Re-select to see new messages.
		c.C(`tag2 select inbox`).OK("tag2")

		// Sort by DATE ascending — should be 2 (2023), 3 (Jul 2023), 1 (2024).
		c.C(`A001 SORT (DATE) UTF-8 ALL`)
		c.S("* SORT 2 3 1")
		c.OK("A001")
	})
}

func TestSortByDateReverse(t *testing.T) {
	runOneToOneTestWithAuth(t, defaultServerOptions(t), func(c *testConnection, s *testSession) {
		c.C(`tag select inbox`).OK("tag")

		c.doAppend("inbox", "Date: Mon, 01 Jan 2024 10:00:00 +0000\r\nFrom: c@test.com\r\nSubject: Third\r\n\r\nbody3").expect("OK")
		c.doAppend("inbox", "Date: Mon, 01 Jan 2023 10:00:00 +0000\r\nFrom: a@test.com\r\nSubject: First\r\n\r\nbody1").expect("OK")
		c.doAppend("inbox", "Date: Mon, 01 Jul 2023 10:00:00 +0000\r\nFrom: b@test.com\r\nSubject: Second\r\n\r\nbody2").expect("OK")

		c.C(`tag2 select inbox`).OK("tag2")

		// Sort by DATE descending.
		c.C(`A001 SORT (REVERSE DATE) UTF-8 ALL`)
		c.S("* SORT 1 3 2")
		c.OK("A001")
	})
}

func TestSortByFrom(t *testing.T) {
	runOneToOneTestWithAuth(t, defaultServerOptions(t), func(c *testConnection, s *testSession) {
		c.C(`tag select inbox`).OK("tag")

		c.doAppend("inbox", "Date: Mon, 01 Jan 2024 10:00:00 +0000\r\nFrom: charlie@test.com\r\nSubject: C\r\n\r\nbody").expect("OK")
		c.doAppend("inbox", "Date: Mon, 01 Jan 2024 10:00:00 +0000\r\nFrom: alice@test.com\r\nSubject: A\r\n\r\nbody").expect("OK")
		c.doAppend("inbox", "Date: Mon, 01 Jan 2024 10:00:00 +0000\r\nFrom: bob@test.com\r\nSubject: B\r\n\r\nbody").expect("OK")

		c.C(`tag2 select inbox`).OK("tag2")

		// Sort by FROM ascending — alice, bob, charlie → 2, 3, 1.
		c.C(`A001 SORT (FROM) UTF-8 ALL`)
		c.S("* SORT 2 3 1")
		c.OK("A001")
	})
}

func TestSortBySubject(t *testing.T) {
	runOneToOneTestWithAuth(t, defaultServerOptions(t), func(c *testConnection, s *testSession) {
		c.C(`tag select inbox`).OK("tag")

		c.doAppend("inbox", "Date: Mon, 01 Jan 2024 10:00:00 +0000\r\nFrom: a@test.com\r\nSubject: Zebra\r\n\r\nbody").expect("OK")
		c.doAppend("inbox", "Date: Mon, 01 Jan 2024 10:00:00 +0000\r\nFrom: b@test.com\r\nSubject: Apple\r\n\r\nbody").expect("OK")
		c.doAppend("inbox", "Date: Mon, 01 Jan 2024 10:00:00 +0000\r\nFrom: c@test.com\r\nSubject: Mango\r\n\r\nbody").expect("OK")

		c.C(`tag2 select inbox`).OK("tag2")

		// Sort by SUBJECT ascending — Apple, Mango, Zebra → 2, 3, 1.
		c.C(`A001 SORT (SUBJECT) UTF-8 ALL`)
		c.S("* SORT 2 3 1")
		c.OK("A001")
	})
}

func TestSortWithSearchCriteria(t *testing.T) {
	runOneToOneTestWithAuth(t, defaultServerOptions(t), func(c *testConnection, s *testSession) {
		c.C(`tag select inbox`).OK("tag")

		c.doAppend("inbox", "Date: Mon, 01 Jan 2024 10:00:00 +0000\r\nFrom: a@test.com\r\nSubject: A\r\n\r\nbody", `\Seen`).expect("OK")
		c.doAppend("inbox", "Date: Mon, 01 Jan 2023 10:00:00 +0000\r\nFrom: b@test.com\r\nSubject: B\r\n\r\nbody").expect("OK")
		c.doAppend("inbox", "Date: Mon, 01 Jul 2023 10:00:00 +0000\r\nFrom: c@test.com\r\nSubject: C\r\n\r\nbody", `\Seen`).expect("OK")

		c.C(`tag2 select inbox`).OK("tag2")

		// Sort by DATE, but only SEEN messages.
		c.C(`A001 SORT (DATE) UTF-8 SEEN`)
		c.S("* SORT 3 1")
		c.OK("A001")
	})
}

func TestSortEmpty(t *testing.T) {
	runOneToOneTestWithAuth(t, defaultServerOptions(t), func(c *testConnection, s *testSession) {
		c.C(`tag select inbox`).OK("tag")

		// Sort on empty mailbox.
		c.C(`A001 SORT (DATE) UTF-8 ALL`)
		c.S("* SORT")
		c.OK("A001")
	})
}

func TestUIDSort(t *testing.T) {
	runOneToOneTestWithAuth(t, defaultServerOptions(t), func(c *testConnection, s *testSession) {
		c.C(`tag select inbox`).OK("tag")

		c.doAppend("inbox", "Date: Mon, 01 Jan 2024 10:00:00 +0000\r\nFrom: c@test.com\r\nSubject: C\r\n\r\nbody").expect("OK")
		c.doAppend("inbox", "Date: Mon, 01 Jan 2023 10:00:00 +0000\r\nFrom: a@test.com\r\nSubject: A\r\n\r\nbody").expect("OK")

		c.C(`tag2 select inbox`).OK("tag2")

		// UID SORT should return UIDs.
		c.C(`A001 UID SORT (DATE) UTF-8 ALL`)
		c.Sx(`\* SORT .*`)
		c.OK("A001")
	})
}

package tests

import (
	"testing"
)

func createSortDisplayMessages(c *testConnection) {
	for _, header := range []string{
		"From: Zoe Adams <adams@test.com>\r\nTo: =?UTF-8?Q?=C3=89ric?= <z@test.com>",
		"From: <carol@test.com>\r\nTo: Bob <bob@test.com>",
		"From: \"Bob, the builder\" <zz@test.com>\r\nTo: alice@test.com",
	} {
		c.doAppend("INBOX", "Date: Mon, 01 Jan 2024 10:00:00 +0000\r\n"+header+"\r\nSubject: display\r\n\r\nbody").expect("OK")
	}

	c.C(`A000 SELECT INBOX`).OK(`A000`)
}

func TestSortDisplayFrom(t *testing.T) {
	runOneToOneTestWithAuth(t, defaultServerOptions(t), func(c *testConnection, _ *testSession) {
		createSortDisplayMessages(c)

		// "bob, the builder", "carol@test.com", "zoe adams"
		c.C(`A001 SORT (DISPLAYFROM) UTF-8 ALL`).S(`* SORT 3 2 1`).OK(`A001`)
		c.C(`A002 SORT (REVERSE DISPLAYFROM) UTF-8 ALL`).S(`* SORT 1 2 3`).OK(`A002`)
		c.C(`A003 UID SORT (displayfrom) UTF-8 ALL`).S(`* SORT 3 2 1`).OK(`A003`)
	})
}

func TestSortDisplayTo(t *testing.T) {
	runOneToOneTestWithAuth(t, defaultServerOptions(t), func(c *testConnection, _ *testSession) {
		createSortDisplayMessages(c)

		// "alice@test.com", "bob", "éric"
		c.C(`A001 SORT (DISPLAYTO) UTF-8 ALL`).S(`* SORT 3 2 1`).OK(`A001`)
		c.C(`A002 SORT (REVERSE DISPLAYTO) UTF-8 ALL`).S(`* SORT 1 2 3`).OK(`A002`)
		c.C(`A003 SORT RETURN (ALL) (DISPLAYTO) UTF-8 ALL`).S(`* ESEARCH (TAG "A003") ALL 3,2,1`).OK(`A003`)
	})
}

func TestSortDisplayWithoutAddressSortsFirst(t *testing.T) {
	runOneToOneTestWithAuth(t, defaultServerOptions(t), func(c *testConnection, _ *testSession) {
		createSortDisplayMessages(c)

		c.doAppend("INBOX", "Date: Mon, 01 Jan 2024 10:00:00 +0000\r\nFrom: Anna <anna@test.com>\r\nSubject: no recipient\r\n\r\nbody").expect("OK")
		c.C(`A001 NOOP`).OK(`A001`)

		c.C(`A002 SORT (DISPLAYTO) UTF-8 ALL`).S(`* SORT 4 3 2 1`).OK(`A002`)
		c.C(`A003 SORT (DISPLAYFROM) UTF-8 ALL`).S(`* SORT 4 3 2 1`).OK(`A003`)
	})
}

func TestSortDisplayFailures(t *testing.T) {
	runOneToOneTestWithAuth(t, defaultServerOptions(t), func(c *testConnection, _ *testSession) {
		createSortDisplayMessages(c)

		c.C(`A001 SORT (DISPLAYCC) UTF-8 ALL`).BAD(`A001`)
		c.C(`A002 SORT (DISPLAY) UTF-8 ALL`).BAD(`A002`)
		c.C(`A003 SORT (DISPLAYFROM) UTF-8`).BAD(`A003`)

		c.C(`A004 CAPABILITY`)
		c.Sx(`\* CAPABILITY .* SORT=DISPLAY `)
		c.OK(`A004`)
	})
}

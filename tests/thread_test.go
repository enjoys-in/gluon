package tests

import (
	"testing"
)

func TestThreadReferences(t *testing.T) {
	runOneToOneTestWithAuth(t, defaultServerOptions(t), func(c *testConnection, s *testSession) {
		c.C(`tag select inbox`).OK("tag")

		// Append a root message.
		c.doAppend("inbox",
			"Date: Mon, 01 Jan 2024 10:00:00 +0000\r\n"+
				"From: alice@test.com\r\n"+
				"Message-Id: <msg1@test.com>\r\n"+
				"Subject: Hello\r\n\r\nbody1").expect("OK")

		// Append a reply to msg1.
		c.doAppend("inbox",
			"Date: Mon, 02 Jan 2024 10:00:00 +0000\r\n"+
				"From: bob@test.com\r\n"+
				"Message-Id: <msg2@test.com>\r\n"+
				"In-Reply-To: <msg1@test.com>\r\n"+
				"References: <msg1@test.com>\r\n"+
				"Subject: Re: Hello\r\n\r\nbody2").expect("OK")

		// Append another independent message.
		c.doAppend("inbox",
			"Date: Mon, 03 Jan 2024 10:00:00 +0000\r\n"+
				"From: charlie@test.com\r\n"+
				"Message-Id: <msg3@test.com>\r\n"+
				"Subject: Different\r\n\r\nbody3").expect("OK")

		c.C(`tag2 select inbox`).OK("tag2")

		// THREAD REFERENCES should group msg1 and msg2 together, msg3 separate.
		c.C(`A001 THREAD REFERENCES UTF-8 ALL`)
		c.Sx(`\* THREAD .*`)
		c.OK("A001")
	})
}

func TestThreadReferencesDeepChain(t *testing.T) {
	runOneToOneTestWithAuth(t, defaultServerOptions(t), func(c *testConnection, s *testSession) {
		c.C(`tag select inbox`).OK("tag")

		// Root message.
		c.doAppend("inbox",
			"Date: Mon, 01 Jan 2024 10:00:00 +0000\r\n"+
				"From: alice@test.com\r\n"+
				"Message-Id: <root@test.com>\r\n"+
				"Subject: Thread test\r\n\r\nbody").expect("OK")

		// Reply to root.
		c.doAppend("inbox",
			"Date: Mon, 02 Jan 2024 10:00:00 +0000\r\n"+
				"From: bob@test.com\r\n"+
				"Message-Id: <reply1@test.com>\r\n"+
				"In-Reply-To: <root@test.com>\r\n"+
				"References: <root@test.com>\r\n"+
				"Subject: Re: Thread test\r\n\r\nbody").expect("OK")

		// Reply to reply1.
		c.doAppend("inbox",
			"Date: Mon, 03 Jan 2024 10:00:00 +0000\r\n"+
				"From: charlie@test.com\r\n"+
				"Message-Id: <reply2@test.com>\r\n"+
				"In-Reply-To: <reply1@test.com>\r\n"+
				"References: <root@test.com> <reply1@test.com>\r\n"+
				"Subject: Re: Re: Thread test\r\n\r\nbody").expect("OK")

		c.C(`tag2 select inbox`).OK("tag2")

		// All three should be in one thread chain.
		c.C(`A001 THREAD REFERENCES UTF-8 ALL`)
		c.Sx(`\* THREAD .*`)
		c.OK("A001")
	})
}

func TestThreadEmpty(t *testing.T) {
	runOneToOneTestWithAuth(t, defaultServerOptions(t), func(c *testConnection, s *testSession) {
		c.C(`tag select inbox`).OK("tag")

		// Empty mailbox.
		c.C(`A001 THREAD REFERENCES UTF-8 ALL`)
		c.S("* THREAD")
		c.OK("A001")
	})
}

func TestThreadWithSearchCriteria(t *testing.T) {
	runOneToOneTestWithAuth(t, defaultServerOptions(t), func(c *testConnection, s *testSession) {
		c.C(`tag select inbox`).OK("tag")

		c.doAppend("inbox",
			"Date: Mon, 01 Jan 2024 10:00:00 +0000\r\n"+
				"From: alice@test.com\r\n"+
				"Message-Id: <msg1@test.com>\r\n"+
				"Subject: Test\r\n\r\nbody", `\Seen`).expect("OK")

		c.doAppend("inbox",
			"Date: Mon, 02 Jan 2024 10:00:00 +0000\r\n"+
				"From: bob@test.com\r\n"+
				"Message-Id: <msg2@test.com>\r\n"+
				"In-Reply-To: <msg1@test.com>\r\n"+
				"Subject: Re: Test\r\n\r\nbody").expect("OK")

		c.C(`tag2 select inbox`).OK("tag2")

		// Thread only SEEN messages — should only include msg1.
		c.C(`A001 THREAD REFERENCES UTF-8 SEEN`)
		c.Sx(`\* THREAD .*`)
		c.OK("A001")
	})
}

func TestUIDThread(t *testing.T) {
	runOneToOneTestWithAuth(t, defaultServerOptions(t), func(c *testConnection, s *testSession) {
		c.C(`tag select inbox`).OK("tag")

		c.doAppend("inbox",
			"Date: Mon, 01 Jan 2024 10:00:00 +0000\r\n"+
				"From: alice@test.com\r\n"+
				"Message-Id: <msg1@test.com>\r\n"+
				"Subject: Test\r\n\r\nbody").expect("OK")

		c.C(`tag2 select inbox`).OK("tag2")

		// UID THREAD should return UIDs.
		c.C(`A001 UID THREAD REFERENCES UTF-8 ALL`)
		c.Sx(`\* THREAD .*`)
		c.OK("A001")
	})
}

func TestThreadOrderedSubject(t *testing.T) {
	runOneToOneTestWithAuth(t, defaultServerOptions(t), func(c *testConnection, s *testSession) {
		c.C(`tag select inbox`).OK("tag")

		c.doAppend("inbox",
			"Date: Mon, 01 Jan 2024 10:00:00 +0000\r\n"+
				"From: alice@test.com\r\n"+
				"Message-Id: <msg1@test.com>\r\n"+
				"Subject: Topic A\r\n\r\nbody").expect("OK")

		c.doAppend("inbox",
			"Date: Mon, 02 Jan 2024 10:00:00 +0000\r\n"+
				"From: bob@test.com\r\n"+
				"Message-Id: <msg2@test.com>\r\n"+
				"Subject: Re: Topic A\r\n\r\nbody").expect("OK")

		c.doAppend("inbox",
			"Date: Mon, 03 Jan 2024 10:00:00 +0000\r\n"+
				"From: charlie@test.com\r\n"+
				"Message-Id: <msg3@test.com>\r\n"+
				"Subject: Topic B\r\n\r\nbody").expect("OK")

		c.C(`tag2 select inbox`).OK("tag2")

		c.C(`A001 THREAD ORDEREDSUBJECT UTF-8 ALL`)
		c.Sx(`\* THREAD .*`)
		c.OK("A001")
	})
}

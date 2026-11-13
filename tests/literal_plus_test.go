package tests

import (
	"testing"
)

func TestLiteralPlusLogin(t *testing.T) {
	runOneToOneTest(t, defaultServerOptions(t), func(c *testConnection, _ *testSession) {
		c.C("A001 LOGIN {4+}\r\nuser {4+}\r\npass")
		c.Sx(`^A001 OK \[CAPABILITY .*LITERAL\+.*\] Logged in`)
	})
}

func TestLiteralPlusLoginFailure(t *testing.T) {
	runOneToOneTest(t, defaultServerOptions(t), func(c *testConnection, _ *testSession) {
		c.C("A001 LOGIN {4+}\r\nuser {5+}\r\nwrong")
		c.Sx(`^A001 NO`)

		c.C("A002 LOGIN {4+}\r\nuser {4+}\r\npass")
		c.Sx(`^A002 OK`)
	})
}

func TestLiteralPlusMixedWithSynchronizingLiteral(t *testing.T) {
	runOneToOneTest(t, defaultServerOptions(t), func(c *testConnection, _ *testSession) {
		c.C("A001 LOGIN {4+}\r\nuser {4}")
		c.Continue()
		c.C("pass")
		c.Sx(`^A001 OK`)
	})
}

func TestLiteralPlusAppend(t *testing.T) {
	runOneToOneTestWithAuth(t, defaultServerOptions(t), func(c *testConnection, _ *testSession) {
		literal := buildRFC5322TestLiteral("To: 1@pm.me\r\nSubject: literal plus\r\n\r\nbody")

		c.Cf("A001 APPEND INBOX {%d+}\r\n%s", len(literal), literal)
		c.Sx(`^A001 OK \[APPENDUID \d+ 1\]`)

		c.Cf("A002 APPEND {5+}\r\nINBOX (\\Seen) {%d+}\r\n%s", len(literal), literal)
		c.Sx(`^A002 OK \[APPENDUID \d+ 2\]`)

		c.C(`A003 STATUS INBOX (MESSAGES UNSEEN)`)
		c.S(`* STATUS "INBOX" (MESSAGES 2 UNSEEN 1)`)
		c.OK(`A003`)
	})
}

func TestLiteralPlusSearch(t *testing.T) {
	runOneToOneTestWithAuth(t, defaultServerOptions(t), func(c *testConnection, _ *testSession) {
		c.doAppend(`INBOX`, buildRFC5322TestLiteral("To: 1@pm.me\r\nSubject: hello world")).expect("OK")
		c.doAppend(`INBOX`, buildRFC5322TestLiteral("To: 2@pm.me\r\nSubject: other")).expect("OK")

		c.C(`A001 SELECT INBOX`).OK(`A001`)

		c.C("A002 SEARCH SUBJECT {11+}\r\nhello world")
		c.S(`* SEARCH 1`)
		c.OK(`A002`)
	})
}

func TestLiteralPlusMailboxNames(t *testing.T) {
	runOneToOneTestWithAuth(t, defaultServerOptions(t), func(c *testConnection, _ *testSession) {
		c.C("A001 CREATE {9+}\r\nMy Folder")
		c.Sx(`^A001 OK`)

		c.C("A002 LIST \"\" {9+}\r\nMy Folder")
		c.S(`* LIST (\HasNoChildren \Unmarked) "/" "My Folder"`)
		c.OK(`A002`)

		c.C("A003 STATUS {9+}\r\nMy Folder (MESSAGES)")
		c.Sx(`^\* STATUS .* \(MESSAGES 0\)`)
		c.OK(`A003`)
	})
}

func TestLiteralPlusRejectedCommandDoesNotRunItsLiteral(t *testing.T) {
	runOneToOneTestWithAuth(t, defaultServerOptions(t), func(c *testConnection, _ *testSession) {
		injected := "X002 CREATE injected\r\n"

		c.Cf("X001 BOGUS {%d+}\r\n%s", len(injected), injected)
		c.Sx(`^X001 BAD`)

		c.C(`X003 LIST "" "*"`)
		c.S(`* LIST (\HasNoChildren \Unmarked) "/" "INBOX"`)
		c.S(`X003 OK LIST`)
	})
}

func TestLiteralPlusRejectedCommandSkipsEveryLiteral(t *testing.T) {
	runOneToOneTestWithAuth(t, defaultServerOptions(t), func(c *testConnection, _ *testSession) {
		first, second := "X002 CREATE one\r\n", "X003 CREATE two\r\n"

		c.Cf("X001 BOGUS {%d+}\r\n%s {%d+}\r\n%s tail", len(first), first, len(second), second)
		c.Sx(`^X001 BAD`)

		c.C(`X004 LIST "" "*"`)
		c.S(`* LIST (\HasNoChildren \Unmarked) "/" "INBOX"`)
		c.S(`X004 OK LIST`)
	})
}

func TestLiteralPlusErrorAfterLiteralKeepsSessionInSync(t *testing.T) {
	runOneToOneTestWithAuth(t, defaultServerOptions(t), func(c *testConnection, _ *testSession) {
		c.C("A001 CREATE {4+}\r\nWork unexpected")
		c.Sx(`BAD .*expected CR`)

		c.C("A002 SELECT {5+}\r\nINBOX")
		c.OK(`A002`)

		c.C(`A003 LIST "" "*"`)
		c.S(`* LIST (\HasNoChildren \Unmarked) "/" "INBOX"`)
		c.S(`A003 OK LIST`)
	})
}

func TestLiteralPlusInvalidMarker(t *testing.T) {
	runOneToOneTest(t, defaultServerOptions(t), func(c *testConnection, _ *testSession) {
		c.C("A001 LOGIN user {4++}")
		c.Sx(`^A001 BAD`)

		c.C("A002 LOGIN user {4-}")
		c.Sx(`^A002 BAD`)

		c.C("A003 LOGIN user pass")
		c.Sx(`^A003 OK`)
	})
}

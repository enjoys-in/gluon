package tests

import (
	"testing"
)

func TestUnauthenticate(t *testing.T) {
	runOneToOneTestWithAuth(t, defaultServerOptions(t), func(c *testConnection, _ *testSession) {
		c.C("A001 CREATE Work").OK("A001")
		c.C("A002 SELECT Work").OK("A002")

		c.C("A003 UNAUTHENTICATE")
		c.S(`A003 OK [CAPABILITY AUTH=PLAIN ID IDLE IMAP4rev1 LITERAL+ SASL-IR STARTTLS] UNAUTHENTICATE`)

		// The session is back in the not authenticated state.
		c.C("A004 SELECT Work").NO("A004")
		c.C("A005 FETCH 1 (FLAGS)").NO("A005")
		c.C("A006 UNAUTHENTICATE").NO("A006")

		c.C("A007 CAPABILITY")
		c.S(`* CAPABILITY AUTH=PLAIN ID IDLE IMAP4rev1 LITERAL+ SASL-IR STARTTLS`)
		c.OK("A007")

		// The same connection can log in again.
		c.C("A008 LOGIN user pass").OK("A008")
		c.C("A009 SELECT Work").OK("A009")
	})
}

func TestUnauthenticateThenLoginAsAnotherUser(t *testing.T) {
	runTest(t, defaultServerOptions(t, withCredentials([]credentials{
		{usernames: []string{"user1"}, password: "pass1"},
		{usernames: []string{"user2"}, password: "pass2"},
	})), []int{1}, func(c map[int]*testConnection, _ *testSession) {
		c[1].C("A001 LOGIN user1 pass1").OK("A001")
		c[1].C("A002 CREATE OnlyForUser1").OK("A002")
		c[1].C("A003 UNAUTHENTICATE").OK("A003")

		c[1].C("A004 AUTHENTICATE PLAIN " + base64AuthString("user2", "pass2")).OK("A004")
		c[1].C(`A005 LIST "" "*"`)
		c[1].S(`* LIST (\HasNoChildren \Unmarked) "/" "INBOX"`)
		c[1].OK("A005")
		c[1].C("A006 SELECT OnlyForUser1").NO("A006")
	})
}

func TestUnauthenticateKeepsOtherSessions(t *testing.T) {
	runManyToOneTestWithAuth(t, defaultServerOptions(t), []int{1, 2}, func(c map[int]*testConnection, _ *testSession) {
		c[1].C("A001 SELECT INBOX").OK("A001")
		c[2].C("B001 SELECT INBOX").OK("B001")

		c[1].C("A002 UNAUTHENTICATE").OK("A002")

		c[2].doAppend("INBOX", buildRFC5322TestLiteral("To: 1@pm.me")).expect("OK")
		c[2].C("B002 NOOP").OK("B002")
		c[2].C("B003 FETCH 1 (UID)").S(`* 1 FETCH (UID 1)`).OK("B003")
	})
}

func TestUnauthenticateBeforeLogin(t *testing.T) {
	runOneToOneTest(t, defaultServerOptions(t), func(c *testConnection, _ *testSession) {
		c.C("A001 UNAUTHENTICATE").NO("A001")
		c.C("A002 LOGIN user pass").OK("A002")
	})
}

func TestUnauthenticateRejectsArguments(t *testing.T) {
	runOneToOneTestWithAuth(t, defaultServerOptions(t), func(c *testConnection, _ *testSession) {
		c.C("A001 UNAUTHENTICATE now").BAD("A001")
		c.C("A002 SELECT INBOX").OK("A002")
	})
}

func TestUnauthenticateAdvertisedAfterLogin(t *testing.T) {
	runOneToOneTest(t, defaultServerOptions(t), func(c *testConnection, _ *testSession) {
		c.C("A001 CAPABILITY")
		c.S(`* CAPABILITY AUTH=PLAIN ID IDLE IMAP4rev1 LITERAL+ SASL-IR STARTTLS`)
		c.OK("A001")

		c.C("A002 LOGIN user pass").OK("A002")

		c.C("A003 CAPABILITY")
		c.Sx(`\* CAPABILITY .* UNAUTHENTICATE`)
		c.OK("A003")
	})
}

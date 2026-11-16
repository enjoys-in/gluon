package tests

import (
	"testing"
)

func TestSASLIRSuccess(t *testing.T) {
	runOneToOneTest(t, defaultServerOptions(t), func(c *testConnection, _ *testSession) {
		c.C("A001 AUTHENTICATE PLAIN " + base64AuthString("user", "pass")).OK("A001")
		c.C("A002 SELECT INBOX").OK("A002")
	})
}

func TestSASLIRAdvertisedWithAuthenticate(t *testing.T) {
	runOneToOneTest(t, defaultServerOptions(t), func(c *testConnection, _ *testSession) {
		c.C("A001 CAPABILITY")
		c.Sx(`\* CAPABILITY .*AUTH=PLAIN.* SASL-IR`)
		c.OK("A001")
	})
}

func TestSASLIRNotAdvertisedWhenAuthenticateIsDisabled(t *testing.T) {
	runOneToOneTest(t, defaultServerOptions(t, withDisableIMAPAuthenticate()), func(c *testConnection, _ *testSession) {
		c.C("A001 CAPABILITY")
		c.S(`* CAPABILITY ID IDLE IMAP4rev1 LITERAL+ STARTTLS`)
		c.OK("A001")

		c.C("A002 AUTHENTICATE PLAIN " + base64AuthString("user", "pass")).BAD("A002")
	})
}

func TestSASLIRWrongPassword(t *testing.T) {
	runOneToOneTest(t, defaultServerOptions(t), func(c *testConnection, _ *testSession) {
		c.C("A001 AUTHENTICATE PLAIN " + base64AuthString("user", "badPass")).NO("A001")
		c.C("A002 AUTHENTICATE PLAIN " + base64AuthString("user", "pass")).OK("A002")
	})
}

func TestSASLIRInvalidInitialResponse(t *testing.T) {
	runOneToOneTest(t, defaultServerOptions(t), func(c *testConnection, _ *testSession) {
		c.C("A001 AUTHENTICATE PLAIN not-base64").BAD("A001")
		c.C("A002 AUTHENTICATE PLAIN =").BAD("A002")
		c.C("A003 AUTHENTICATE PLAIN " + base64AuthString("user", "pass") + " extra").BAD("A003")
		c.C("A004 AUTHENTICATE LOGIN " + base64AuthString("user", "pass")).BAD("A004")

		c.C("A005 AUTHENTICATE PLAIN")
		c.S("+")
		c.C(base64AuthString("user", "pass")).OK("A005")
	})
}

func TestSASLIRWhenAlreadyAuthenticated(t *testing.T) {
	runOneToOneTestWithAuth(t, defaultServerOptions(t), func(c *testConnection, _ *testSession) {
		c.C("A001 AUTHENTICATE PLAIN " + base64AuthString("user", "pass")).BAD("A001")
	})
}

package tests

import (
	"testing"
)

func TestNamespace(t *testing.T) {
	runOneToOneTestWithAuth(t, defaultServerOptions(t), func(c *testConnection, _ *testSession) {
		c.C("A001 NAMESPACE")
		c.S(`* NAMESPACE (("" "/")) NIL NIL`)
		c.S("A001 OK NAMESPACE")
	})
}

func TestNamespaceLowerCase(t *testing.T) {
	runOneToOneTestWithAuth(t, defaultServerOptions(t), func(c *testConnection, _ *testSession) {
		c.C("A001 namespace")
		c.S(`* NAMESPACE (("" "/")) NIL NIL`)
		c.S("A001 OK NAMESPACE")
	})
}

func TestNamespaceCustomDelimiter(t *testing.T) {
	runOneToOneTestWithAuth(t, defaultServerOptions(t, withDelimiter(".")), func(c *testConnection, _ *testSession) {
		c.C("A001 NAMESPACE")
		c.S(`* NAMESPACE (("" ".")) NIL NIL`)
		c.S("A001 OK NAMESPACE")
	})
}

func TestNamespaceDelimiterMatchesList(t *testing.T) {
	runOneToOneTestWithAuth(t, defaultServerOptions(t, withDelimiter(".")), func(c *testConnection, _ *testSession) {
		c.C(`A001 LIST "" ""`)
		c.S(`* LIST (\Noselect) "." ""`)
		c.OK("A001")

		c.C("A002 NAMESPACE")
		c.S(`* NAMESPACE (("" ".")) NIL NIL`)
		c.S("A002 OK NAMESPACE")
	})
}

func TestNamespaceInSelectedState(t *testing.T) {
	runOneToOneTestWithAuth(t, defaultServerOptions(t), func(c *testConnection, _ *testSession) {
		c.C("A001 SELECT INBOX").OK("A001")

		c.C("A002 NAMESPACE")
		c.S(`* NAMESPACE (("" "/")) NIL NIL`)
		c.S("A002 OK NAMESPACE")
	})
}

func TestNamespaceRepeated(t *testing.T) {
	runOneToOneTestWithAuth(t, defaultServerOptions(t), func(c *testConnection, _ *testSession) {
		for _, tag := range []string{"A001", "A002", "A003"} {
			c.C(tag + " NAMESPACE")
			c.S(`* NAMESPACE (("" "/")) NIL NIL`)
			c.S(tag + " OK NAMESPACE")
		}
	})
}

func TestNamespaceNotAuthenticated(t *testing.T) {
	runOneToOneTest(t, defaultServerOptions(t), func(c *testConnection, _ *testSession) {
		c.C("A001 NAMESPACE")
		c.S("A001 NO session is not authenticated")

		c.C("A002 login user pass")
		c.OK("A002")

		c.C("A003 NAMESPACE")
		c.S(`* NAMESPACE (("" "/")) NIL NIL`)
		c.S("A003 OK NAMESPACE")
	})
}

func TestNamespaceRejectsArguments(t *testing.T) {
	runOneToOneTestWithAuth(t, defaultServerOptions(t), func(c *testConnection, _ *testSession) {
		c.C("A001 NAMESPACE INBOX")
		c.Sx(`BAD .*expected CR`)

		c.C(`A002 NAMESPACE ""`)
		c.Sx(`BAD .*expected CR`)

		c.C("A003 NAMESPACE")
		c.S(`* NAMESPACE (("" "/")) NIL NIL`)
		c.S("A003 OK NAMESPACE")
	})
}

func TestNamespaceNotAdvertisedBeforeLogin(t *testing.T) {
	runOneToOneTest(t, defaultServerOptions(t), func(c *testConnection, _ *testSession) {
		c.C("A001 CAPABILITY")
		c.S(`* CAPABILITY AUTH=PLAIN ID IDLE IMAP4rev1 LITERAL+ STARTTLS`)
		c.OK("A001")

		c.C("A002 login user pass")
		c.OK("A002")

		c.C("A003 CAPABILITY")
		c.Sx(`\* CAPABILITY .* NAMESPACE .*`)
		c.OK("A003")
	})
}

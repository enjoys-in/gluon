package tests

import (
	"testing"

	"github.com/ProtonMail/gluon/imap/command"
	"github.com/stretchr/testify/require"
)

func TestEnable(t *testing.T) {
	runOneToOneTestWithAuth(t, defaultServerOptions(t), func(c *testConnection, _ *testSession) {
		c.C(`A001 ENABLE UTF8=ACCEPT`).S(`* ENABLED UTF8=ACCEPT`).OK(`A001`)

		// The response lists what this command enabled, which is nothing the second time.
		c.C(`A002 ENABLE UTF8=ACCEPT`).S(`* ENABLED`).OK(`A002`)
	})
}

func TestEnableIgnoresWhatCannotBeEnabled(t *testing.T) {
	runOneToOneTestWithAuth(t, defaultServerOptions(t), func(c *testConnection, _ *testSession) {
		// Unknown names and capabilities that need no enabling are ignored, as RFC 5161 asks.
		c.C(`A001 ENABLE CONDSTORE X-GOOD-IDEA`).S(`* ENABLED`).OK(`A001`)
		c.C(`A002 ENABLE IDLE ESEARCH`).S(`* ENABLED`).OK(`A002`)

		c.C(`A003 enable x-good-idea utf8=accept QRESYNC`).S(`* ENABLED UTF8=ACCEPT`).OK(`A003`)
	})
}

func TestEnableDoesNotChangeTheCapabilities(t *testing.T) {
	runOneToOneTestWithAuth(t, defaultServerOptions(t), func(c *testConnection, _ *testSession) {
		c.C(`A001 CAPABILITY`)
		before := string(c.read())
		c.OK(`A001`)

		require.Contains(t, before, " ENABLE ")
		require.Contains(t, before, " UTF8=ACCEPT")

		c.C(`A002 ENABLE UTF8=ACCEPT`).S(`* ENABLED UTF8=ACCEPT`).OK(`A002`)

		c.C(`A003 CAPABILITY`)
		require.Equal(t, before, string(c.read()))
		c.OK(`A003`)
	})
}

func TestEnableFailures(t *testing.T) {
	runOneToOneTest(t, defaultServerOptions(t), func(c *testConnection, _ *testSession) {
		// ENABLE is for the authenticated state.
		c.C(`A001 ENABLE UTF8=ACCEPT`).NO(`A001`)

		c.C(`A002 CAPABILITY`)
		c.S(`* CAPABILITY AUTH=PLAIN ID IDLE IMAP4rev1 LITERAL+ SASL-IR STARTTLS`)
		c.OK(`A002`)

		c.C(`A003 LOGIN user pass`).OK(`A003`)

		c.C(`A004 ENABLE`).BAD(`A004`)
		c.C(`A005 ENABLE `).BAD(`A005`)
		c.C(`A006 ENABLE (UTF8=ACCEPT)`).BAD(`A006`)
		c.C(`A007 ENABLE UTF8=ACCEPT `).BAD(`A007`)

		// Nothing was enabled by the commands that failed.
		c.C(`A008 ENABLE UTF8=ACCEPT`).S(`* ENABLED UTF8=ACCEPT`).OK(`A008`)
	})
}

func TestParserEnable(t *testing.T) {
	cmd, err := parseCommand(`tag ENABLE CONDSTORE UTF8=ACCEPT`)
	require.NoError(t, err)
	require.Equal(t, command.Command{Tag: "tag", Payload: &command.Enable{Capabilities: []string{"CONDSTORE", "UTF8=ACCEPT"}}}, cmd)
}

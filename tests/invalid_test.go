package tests

import (
	"testing"
)

func TestInvalidIMAPCommandBadTag(t *testing.T) {
	runOneToOneTestWithAuth(t, defaultServerOptions(t), func(c *testConnection, s *testSession) {
		c.C("A006 RANDOMGIBBERISHTHATDOESNOTMAKEAVALIDIMAPCOMMAND").BAD("A006")
	})
}

func TestInvalidIMAPCommandTrailingArgumentsKeepTheTag(t *testing.T) {
	runOneToOneTestWithAuth(t, defaultServerOptions(t), func(c *testConnection, s *testSession) {
		c.C("A001 NOOP junk").BAD("A001")
		c.C("A002 CAPABILITY junk").BAD("A002")
		c.C("A003 STARTTLS junk").BAD("A003")
		c.C("A004 NOOP").OK("A004")
	})
}

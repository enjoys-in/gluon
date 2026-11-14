package tests

import (
	"testing"

	"github.com/ProtonMail/gluon/imap"
)

func TestListChildren(t *testing.T) {
	runOneToOneTestWithAuth(t, defaultServerOptions(t), func(c *testConnection, _ *testSession) {
		c.C(`A001 CREATE Work/Reports/2026`).OK(`A001`)
		c.C(`A002 CREATE Personal`).OK(`A002`)

		c.C(`A003 LIST "" "*"`)
		c.S(
			`* LIST (\HasNoChildren \Unmarked) "/" "INBOX"`,
			`* LIST (\HasNoChildren \Unmarked) "/" "Personal"`,
			`* LIST (\HasChildren \Unmarked) "/" "Work"`,
			`* LIST (\HasChildren \Unmarked) "/" "Work/Reports"`,
			`* LIST (\HasNoChildren \Unmarked) "/" "Work/Reports/2026"`,
		)
		c.OK(`A003`)

		c.C(`A004 LIST "" "%"`)
		c.S(
			`* LIST (\HasNoChildren \Unmarked) "/" "INBOX"`,
			`* LIST (\HasNoChildren \Unmarked) "/" "Personal"`,
			`* LIST (\HasChildren \Unmarked) "/" "Work"`,
		)
		c.OK(`A004`)
	})
}

func TestListChildrenFollowsCreateAndDelete(t *testing.T) {
	runOneToOneTestWithAuth(t, defaultServerOptions(t), func(c *testConnection, _ *testSession) {
		c.C(`A001 CREATE Work`).OK(`A001`)
		c.C(`A002 LIST "" Work`).S(`* LIST (\HasNoChildren \Unmarked) "/" "Work"`).OK(`A002`)

		c.C(`A003 CREATE Work/Reports`).OK(`A003`)
		c.C(`A004 LIST "" Work`).S(`* LIST (\HasChildren \Unmarked) "/" "Work"`).OK(`A004`)

		c.C(`A005 DELETE Work/Reports`).OK(`A005`)
		c.C(`A006 LIST "" Work`).S(`* LIST (\HasNoChildren \Unmarked) "/" "Work"`).OK(`A006`)
	})
}

func TestListChildrenOfDeletedParent(t *testing.T) {
	runOneToOneTestWithAuth(t, defaultServerOptions(t), func(c *testConnection, _ *testSession) {
		c.C(`A001 CREATE Work/Reports`).OK(`A001`)
		c.C(`A002 DELETE Work`).OK(`A002`)

		c.C(`A003 LIST "" "*"`)
		c.S(
			`* LIST (\HasNoChildren \Unmarked) "/" "INBOX"`,
			`* LIST (\HasChildren \Noselect) "/" "Work"`,
			`* LIST (\HasNoChildren \Unmarked) "/" "Work/Reports"`,
		)
		c.OK(`A003`)
	})
}

func TestListChildrenIgnoresNameWithSamePrefix(t *testing.T) {
	runOneToOneTestWithAuth(t, defaultServerOptions(t), func(c *testConnection, _ *testSession) {
		c.C(`A001 CREATE Work`).OK(`A001`)
		c.C(`A002 CREATE Workshop`).OK(`A002`)

		c.C(`A003 LIST "" "Work*"`)
		c.S(
			`* LIST (\HasNoChildren \Unmarked) "/" "Work"`,
			`* LIST (\HasNoChildren \Unmarked) "/" "Workshop"`,
		)
		c.OK(`A003`)
	})
}

func TestListChildrenIgnoresHiddenChild(t *testing.T) {
	runOneToOneTestWithAuth(t, defaultServerOptions(t), func(c *testConnection, s *testSession) {
		s.mailboxCreated("user", []string{"Parent"})
		hidden := s.mailboxCreated("user", []string{"Parent", "Hidden"})

		c.C(`A001 LIST "" Parent`).S(`* LIST (\HasChildren \Unmarked) "/" "Parent"`).OK(`A001`)

		s.conns[s.userIDs["user"]].SetMailboxVisibility(hidden, imap.Hidden)

		c.C(`A002 LIST "" Parent`).S(`* LIST (\HasNoChildren \Unmarked) "/" "Parent"`).OK(`A002`)
	})
}

func TestListChildrenNotGivenToNoinferiors(t *testing.T) {
	runOneToOneTestWithAuth(t, defaultServerOptions(t), func(c *testConnection, s *testSession) {
		s.mailboxCreatedCustom("user", []string{"Leaf"}, defaultFlags, defaultPermanentFlags, imap.NewFlagSet(imap.AttrNoInferiors))

		c.C(`A001 LIST "" Leaf`).S(`* LIST (\Noinferiors \Unmarked) "/" "Leaf"`).OK(`A001`)
	})
}

func TestListChildrenNotGivenToHierarchyRoot(t *testing.T) {
	runOneToOneTestWithAuth(t, defaultServerOptions(t), func(c *testConnection, _ *testSession) {
		c.C(`A001 CREATE Work/Reports`).OK(`A001`)

		c.C(`A002 LIST "" ""`).S(`* LIST (\Noselect) "/" ""`).OK(`A002`)
		c.C(`A003 LIST "Work/Reports" ""`).S(`* LIST (\Noselect) "/" "Work/"`).OK(`A003`)
	})
}

func TestLsubHasNoChildAttributes(t *testing.T) {
	runOneToOneTestWithAuth(t, defaultServerOptions(t), func(c *testConnection, _ *testSession) {
		c.C(`A001 CREATE Work/Reports`).OK(`A001`)

		c.C(`A002 LSUB "" "*"`)
		c.S(
			`* LSUB (\Unmarked) "/" "INBOX"`,
			`* LSUB (\Unmarked) "/" "Work"`,
			`* LSUB (\Unmarked) "/" "Work/Reports"`,
		)
		c.OK(`A002`)
	})
}

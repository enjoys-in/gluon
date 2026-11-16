package tests

import (
	"testing"

	"github.com/ProtonMail/gluon/imap"
)

func TestListExtendedWithoutOptionsMatchesList(t *testing.T) {
	runOneToOneTestWithAuth(t, defaultServerOptions(t), func(c *testConnection, _ *testSession) {
		c.C(`A001 CREATE Work/Reports`).OK(`A001`)

		want := []string{
			`* LIST (\HasNoChildren \Unmarked) "/" "INBOX"`,
			`* LIST (\HasChildren \Unmarked) "/" "Work"`,
			`* LIST (\HasNoChildren \Unmarked) "/" "Work/Reports"`,
		}

		c.C(`A002 LIST "" "*"`).S(want...).OK(`A002`)
		c.C(`A003 LIST () "" "*"`).S(want...).OK(`A003`)
		c.C(`A004 LIST "" "*" RETURN ()`).S(want...).OK(`A004`)
		c.C(`A005 LIST (REMOTE) "" "*" RETURN (CHILDREN)`).S(want...).OK(`A005`)
		c.C(`A006 list () "" ("*") return (children special-use)`).S(want...).OK(`A006`)
	})
}

func TestListExtendedIsSortedByName(t *testing.T) {
	runOneToOneTestWithAuth(t, defaultServerOptions(t), func(c *testConnection, _ *testSession) {
		c.C(`A001 CREATE Zebra`).OK(`A001`)
		c.C(`A002 CREATE Apple`).OK(`A002`)

		c.C(`A003 LIST "" "*" RETURN (CHILDREN)`)
		c.S(`* LIST (\HasNoChildren \Unmarked) "/" "Apple"`)
		c.S(`* LIST (\HasNoChildren \Unmarked) "/" "INBOX"`)
		c.S(`* LIST (\HasNoChildren \Unmarked) "/" "Zebra"`)
		c.S(`A003 OK LIST`)
	})
}

func TestListExtendedMultiplePatterns(t *testing.T) {
	runOneToOneTestWithAuth(t, defaultServerOptions(t), func(c *testConnection, _ *testSession) {
		c.C(`A001 CREATE Work/Reports`).OK(`A001`)
		c.C(`A002 CREATE Personal`).OK(`A002`)

		c.C(`A003 LIST "" ("INBOX" "Work/%")`)
		c.S(
			`* LIST (\HasNoChildren \Unmarked) "/" "INBOX"`,
			`* LIST (\HasNoChildren \Unmarked) "/" "Work/Reports"`,
		)
		c.S(`A003 OK LIST`)

		c.C(`A004 LIST "" (INBOX "inbox" INBOX)`)
		c.S(`* LIST (\HasNoChildren \Unmarked) "/" "INBOX"`)
		c.S(`A004 OK LIST`)

		c.C(`A005 LIST "Work/" (% Reports)`)
		c.S(`* LIST (\HasNoChildren \Unmarked) "/" "Work/Reports"`)
		c.S(`A005 OK LIST`)

		c.C(`A006 LIST "" (Missing AlsoMissing)`)
		c.S(`A006 OK LIST`)
	})
}

func TestListExtendedReturnSubscribed(t *testing.T) {
	runOneToOneTestWithAuth(t, defaultServerOptions(t), func(c *testConnection, _ *testSession) {
		c.C(`A001 CREATE Work`).OK(`A001`)
		c.C(`A002 CREATE Personal`).OK(`A002`)
		c.C(`A003 UNSUBSCRIBE Personal`).OK(`A003`)

		c.C(`A004 LIST "" "*" RETURN (SUBSCRIBED)`)
		c.S(
			`* LIST (\HasNoChildren \Subscribed \Unmarked) "/" "INBOX"`,
			`* LIST (\HasNoChildren \Unmarked) "/" "Personal"`,
			`* LIST (\HasNoChildren \Subscribed \Unmarked) "/" "Work"`,
		)
		c.S(`A004 OK LIST`)

		c.C(`A005 LIST "" "*"`)
		c.S(
			`* LIST (\HasNoChildren \Unmarked) "/" "INBOX"`,
			`* LIST (\HasNoChildren \Unmarked) "/" "Personal"`,
			`* LIST (\HasNoChildren \Unmarked) "/" "Work"`,
		)
		c.S(`A005 OK LIST`)
	})
}

func TestListExtendedSelectSubscribed(t *testing.T) {
	runOneToOneTestWithAuth(t, defaultServerOptions(t), func(c *testConnection, _ *testSession) {
		c.C(`A001 CREATE Work`).OK(`A001`)
		c.C(`A002 CREATE Personal`).OK(`A002`)
		c.C(`A003 UNSUBSCRIBE Personal`).OK(`A003`)

		c.C(`A004 LIST (SUBSCRIBED) "" "*"`)
		c.S(
			`* LIST (\HasNoChildren \Subscribed \Unmarked) "/" "INBOX"`,
			`* LIST (\HasNoChildren \Subscribed \Unmarked) "/" "Work"`,
		)
		c.S(`A004 OK LIST`)

		c.C(`A005 SUBSCRIBE Personal`).OK(`A005`)

		c.C(`A006 LIST (SUBSCRIBED) "" "P*"`)
		c.S(`* LIST (\HasNoChildren \Subscribed \Unmarked) "/" "Personal"`)
		c.S(`A006 OK LIST`)
	})
}

func TestListExtendedSelectSubscribedWithNothingSubscribed(t *testing.T) {
	runOneToOneTestWithAuth(t, defaultServerOptions(t), func(c *testConnection, _ *testSession) {
		c.C(`A001 UNSUBSCRIBE INBOX`).OK(`A001`)

		c.C(`A002 LIST (SUBSCRIBED) "" "*"`)
		c.S(`A002 OK LIST`)

		c.C(`A003 LIST (SUBSCRIBED RECURSIVEMATCH) "" "*"`)
		c.S(`A003 OK LIST`)
	})
}

func TestListExtendedSelectSubscribedReportsDeletedMailbox(t *testing.T) {
	runOneToOneTestWithAuth(t, defaultServerOptions(t), func(c *testConnection, _ *testSession) {
		c.C(`A001 CREATE Gone`).OK(`A001`)
		c.C(`A002 DELETE Gone`).OK(`A002`)

		c.C(`A003 LIST (SUBSCRIBED) "" "*"`)
		c.S(
			`* LIST (\HasNoChildren \NonExistent \Subscribed) "/" "Gone"`,
			`* LIST (\HasNoChildren \Subscribed \Unmarked) "/" "INBOX"`,
		)
		c.S(`A003 OK LIST`)

		c.C(`A004 LIST "" "*" RETURN (SUBSCRIBED)`)
		c.S(`* LIST (\HasNoChildren \Subscribed \Unmarked) "/" "INBOX"`)
		c.S(`A004 OK LIST`)

		c.C(`A005 UNSUBSCRIBE Gone`).OK(`A005`)

		c.C(`A006 LIST (SUBSCRIBED) "" "*"`)
		c.S(`* LIST (\HasNoChildren \Subscribed \Unmarked) "/" "INBOX"`)
		c.S(`A006 OK LIST`)
	})
}

func TestListExtendedRecursiveMatch(t *testing.T) {
	runOneToOneTestWithAuth(t, defaultServerOptions(t), func(c *testConnection, _ *testSession) {
		c.C(`A001 CREATE Fruit/Banana`).OK(`A001`)
		c.C(`A002 CREATE Vegetable`).OK(`A002`)
		c.C(`A003 UNSUBSCRIBE Fruit`).OK(`A003`)
		c.C(`A004 UNSUBSCRIBE Vegetable`).OK(`A004`)

		c.C(`A005 LIST (SUBSCRIBED) "" "%"`)
		c.S(`* LIST (\HasNoChildren \Subscribed \Unmarked) "/" "INBOX"`)
		c.S(`A005 OK LIST`)

		c.C(`A006 LIST (SUBSCRIBED RECURSIVEMATCH) "" "%"`)
		c.S(
			`* LIST (\HasChildren \Unmarked) "/" "Fruit" ("CHILDINFO" ("SUBSCRIBED"))`,
			`* LIST (\HasNoChildren \Subscribed \Unmarked) "/" "INBOX"`,
		)
		c.S(`A006 OK LIST`)

		c.C(`A007 LIST (SUBSCRIBED RECURSIVEMATCH) "" "*"`)
		c.S(
			`* LIST (\HasChildren \Unmarked) "/" "Fruit" ("CHILDINFO" ("SUBSCRIBED"))`,
			`* LIST (\HasNoChildren \Subscribed \Unmarked) "/" "Fruit/Banana"`,
			`* LIST (\HasNoChildren \Subscribed \Unmarked) "/" "INBOX"`,
		)
		c.S(`A007 OK LIST`)
	})
}

func TestListExtendedRecursiveMatchWithSubscribedParent(t *testing.T) {
	runOneToOneTestWithAuth(t, defaultServerOptions(t), func(c *testConnection, _ *testSession) {
		c.C(`A001 CREATE Fruit/Banana`).OK(`A001`)

		c.C(`A002 LIST (SUBSCRIBED RECURSIVEMATCH) "" "%"`)
		c.S(
			`* LIST (\HasChildren \Subscribed \Unmarked) "/" "Fruit" ("CHILDINFO" ("SUBSCRIBED"))`,
			`* LIST (\HasNoChildren \Subscribed \Unmarked) "/" "INBOX"`,
		)
		c.S(`A002 OK LIST`)
	})
}

func TestListExtendedRecursiveMatchWithDeletedParent(t *testing.T) {
	runOneToOneTestWithAuth(t, defaultServerOptions(t), func(c *testConnection, _ *testSession) {
		c.C(`A001 CREATE Fruit/Banana`).OK(`A001`)
		c.C(`A002 UNSUBSCRIBE Fruit`).OK(`A002`)
		c.C(`A003 DELETE Fruit`).OK(`A003`)

		c.C(`A004 LIST (SUBSCRIBED RECURSIVEMATCH) "" "%"`)
		c.S(
			`* LIST (\HasChildren \NonExistent) "/" "Fruit" ("CHILDINFO" ("SUBSCRIBED"))`,
			`* LIST (\HasNoChildren \Subscribed \Unmarked) "/" "INBOX"`,
		)
		c.S(`A004 OK LIST`)
	})
}

func TestListExtendedSpecialUse(t *testing.T) {
	runOneToOneTestWithAuth(t, defaultServerOptions(t), func(c *testConnection, s *testSession) {
		s.mailboxCreatedWithAttributes("user", []string{"Sent Items"}, imap.NewFlagSet(imap.AttrSent))
		s.mailboxCreatedWithAttributes("user", []string{"Bin"}, imap.NewFlagSet(imap.AttrTrash))
		s.mailboxCreatedWithAttributes("user", []string{"Plain"}, imap.NewFlagSet())

		c.C(`A001 LIST (SPECIAL-USE) "" "*"`)
		c.S(
			`* LIST (\HasNoChildren \Trash \Unmarked) "/" "Bin"`,
			`* LIST (\HasNoChildren \Sent \Unmarked) "/" "Sent Items"`,
		)
		c.S(`A001 OK LIST`)

		c.C(`A002 LIST "" "*" RETURN (SPECIAL-USE)`)
		c.S(
			`* LIST (\HasNoChildren \Trash \Unmarked) "/" "Bin"`,
			`* LIST (\HasNoChildren \Unmarked) "/" "INBOX"`,
			`* LIST (\HasNoChildren \Unmarked) "/" "Plain"`,
			`* LIST (\HasNoChildren \Sent \Unmarked) "/" "Sent Items"`,
		)
		c.S(`A002 OK LIST`)

		c.C(`A003 LIST (SPECIAL-USE) "" "S*" RETURN (SPECIAL-USE)`)
		c.S(`* LIST (\HasNoChildren \Sent \Unmarked) "/" "Sent Items"`)
		c.S(`A003 OK LIST`)

		c.C(`A004 LIST (SPECIAL-USE) "" "Plain"`)
		c.S(`A004 OK LIST`)
	})
}

func TestListExtendedSpecialUseAndSubscribed(t *testing.T) {
	runOneToOneTestWithAuth(t, defaultServerOptions(t), func(c *testConnection, s *testSession) {
		s.mailboxCreatedWithAttributes("user", []string{"Sent Items"}, imap.NewFlagSet(imap.AttrSent))
		s.mailboxCreatedWithAttributes("user", []string{"Bin"}, imap.NewFlagSet(imap.AttrTrash))

		c.C(`A001 SUBSCRIBE "Sent Items"`).OK(`A001`)
		c.C(`A002 UNSUBSCRIBE Bin`).OK(`A002`)

		c.C(`A003 LIST (SPECIAL-USE SUBSCRIBED) "" "*"`)
		c.S(`* LIST (\HasNoChildren \Sent \Subscribed \Unmarked) "/" "Sent Items"`)
		c.S(`A003 OK LIST`)
	})
}

func TestListExtendedInvalid(t *testing.T) {
	runOneToOneTestWithAuth(t, defaultServerOptions(t), func(c *testConnection, _ *testSession) {
		c.C(`A001 LIST (RECURSIVEMATCH) "" "*"`)
		c.Sx(`^A001 BAD .*RECURSIVEMATCH requires the SUBSCRIBED selection option`)

		c.C(`A002 LIST (REMOTE RECURSIVEMATCH) "" "*"`)
		c.Sx(`^A002 BAD .*RECURSIVEMATCH requires the SUBSCRIBED selection option`)

		c.C(`A003 LIST (BOGUS) "" "*"`)
		c.Sx(`^A003 BAD .*unknown list selection option 'BOGUS'`)

		c.C(`A004 LIST "" "*" RETURN (BOGUS)`)
		c.Sx(`^A004 BAD .*unknown list return option 'BOGUS'`)

		c.C(`A005 LIST "" ()`)
		c.Sx(`^A005 BAD`)

		c.C(`A006 LIST "" "*" CHILDREN`)
		c.Sx(`^A006 BAD .*expected RETURN`)

		c.C(`A007 LIST "" "*" RETURN CHILDREN`)
		c.Sx(`^A007 BAD`)

		c.C(`A008 LIST "" "*" RETURN (CHILDREN)`)
		c.S(`* LIST (\HasNoChildren \Unmarked) "/" "INBOX"`)
		c.S(`A008 OK LIST`)
	})
}

func TestListExtendedNotAuthenticated(t *testing.T) {
	runOneToOneTest(t, defaultServerOptions(t), func(c *testConnection, _ *testSession) {
		c.C(`A001 LIST (SUBSCRIBED) "" "*" RETURN (STATUS (MESSAGES))`)
		c.S(`A001 NO session is not authenticated`)
	})
}

func TestListStatus(t *testing.T) {
	runOneToOneTestWithAuth(t, defaultServerOptions(t), func(c *testConnection, _ *testSession) {
		c.C(`A001 CREATE Work`).OK(`A001`)

		c.doAppend(`INBOX`, buildRFC5322TestLiteral(`To: 1@pm.me`)).expect("OK")
		c.doAppend(`INBOX`, buildRFC5322TestLiteral(`To: 2@pm.me`), `\Seen`).expect("OK")
		c.doAppend(`Work`, buildRFC5322TestLiteral(`To: 3@pm.me`)).expect("OK")

		c.C(`A002 LIST "" "*" RETURN (STATUS (MESSAGES UNSEEN))`)
		c.S(`* LIST (\HasNoChildren \Marked) "/" "INBOX"`)
		c.S(`* STATUS "INBOX" (MESSAGES 2 UNSEEN 1)`)
		c.S(`* LIST (\HasNoChildren \Marked) "/" "Work"`)
		c.S(`* STATUS "Work" (MESSAGES 1 UNSEEN 1)`)
		c.S(`A002 OK LIST`)

		c.C(`A003 STATUS Work (MESSAGES UNSEEN)`)
		c.S(`* STATUS "Work" (MESSAGES 1 UNSEEN 1)`)
		c.OK(`A003`)
	})
}

func TestListStatusAllItems(t *testing.T) {
	runOneToOneTestWithAuth(t, defaultServerOptions(t), func(c *testConnection, _ *testSession) {
		c.doAppend(`INBOX`, buildRFC5322TestLiteral(`To: 1@pm.me`)).expect("OK")

		c.C(`A001 LIST "" INBOX RETURN (STATUS (MESSAGES RECENT UIDNEXT UIDVALIDITY UNSEEN))`)
		c.S(`* LIST (\HasNoChildren \Marked) "/" "INBOX"`)
		c.Sx(`^\* STATUS "INBOX" \(MESSAGES 1 RECENT 1 UIDNEXT 2 UIDVALIDITY \d+ UNSEEN 1\)`)
		c.S(`A001 OK LIST`)
	})
}

func TestListStatusOnlyForMatchedMailboxes(t *testing.T) {
	runOneToOneTestWithAuth(t, defaultServerOptions(t), func(c *testConnection, _ *testSession) {
		c.C(`A001 CREATE Work`).OK(`A001`)

		c.C(`A002 LIST "" ("Work" "Missing") RETURN (STATUS (MESSAGES))`)
		c.S(`* LIST (\HasNoChildren \Unmarked) "/" "Work"`)
		c.S(`* STATUS "Work" (MESSAGES 0)`)
		c.S(`A002 OK LIST`)

		c.C(`A003 LIST "" "Missing" RETURN (STATUS (MESSAGES))`)
		c.S(`A003 OK LIST`)
	})
}

func TestListStatusSkipsUnselectableNames(t *testing.T) {
	runOneToOneTestWithAuth(t, defaultServerOptions(t), func(c *testConnection, _ *testSession) {
		c.C(`A001 CREATE Work/Reports`).OK(`A001`)
		c.C(`A002 DELETE Work`).OK(`A002`)
		c.C(`A003 CREATE Gone`).OK(`A003`)
		c.C(`A004 DELETE Gone`).OK(`A004`)

		c.C(`A005 LIST "" "*" RETURN (STATUS (MESSAGES))`)
		c.S(`* LIST (\HasNoChildren \Unmarked) "/" "INBOX"`)
		c.S(`* STATUS "INBOX" (MESSAGES 0)`)
		c.S(`* LIST (\HasChildren \Noselect) "/" "Work"`)
		c.S(`* LIST (\HasNoChildren \Unmarked) "/" "Work/Reports"`)
		c.S(`* STATUS "Work/Reports" (MESSAGES 0)`)
		c.S(`A005 OK LIST`)

		c.C(`A006 LIST (SUBSCRIBED) "" "*" RETURN (STATUS (MESSAGES))`)
		c.S(`* LIST (\HasNoChildren \NonExistent \Subscribed) "/" "Gone"`)
		c.S(`* LIST (\HasNoChildren \Subscribed \Unmarked) "/" "INBOX"`)
		c.S(`* STATUS "INBOX" (MESSAGES 0)`)
		c.S(`* LIST (\HasChildren \NonExistent \Subscribed) "/" "Work"`)
		c.S(`* LIST (\HasNoChildren \Subscribed \Unmarked) "/" "Work/Reports"`)
		c.S(`* STATUS "Work/Reports" (MESSAGES 0)`)
		c.S(`A006 OK LIST`)

		c.C(`A007 LIST "" "" RETURN (STATUS (MESSAGES))`)
		c.S(`* LIST (\Noselect) "/" ""`)
		c.S(`A007 OK LIST`)
	})
}

func TestListStatusInSelectedState(t *testing.T) {
	runOneToOneTestWithAuth(t, defaultServerOptions(t), func(c *testConnection, _ *testSession) {
		c.doAppend(`INBOX`, buildRFC5322TestLiteral(`To: 1@pm.me`)).expect("OK")

		c.C(`A001 SELECT INBOX`).OK(`A001`)

		c.C(`A002 LIST "" "*" RETURN (STATUS (MESSAGES UNSEEN))`)
		c.S(`* LIST (\HasNoChildren \Unmarked) "/" "INBOX"`)
		c.S(`* STATUS "INBOX" (MESSAGES 1 UNSEEN 1)`)
		c.S(`A002 OK LIST`)
	})
}

func TestListStatusEncodesMailboxNameLikeList(t *testing.T) {
	runOneToOneTestWithAuth(t, defaultServerOptions(t), func(c *testConnection, s *testSession) {
		s.mailboxCreated("user", []string{"mbox-öüäëæøå"})
		s.flush("user")

		c.C(`A001 LIST "" "mbox*" RETURN (STATUS (MESSAGES))`)
		c.S(`* LIST (\HasNoChildren \Unmarked) "/" "mbox-&APYA,ADkAOsA5gD4AOU-"`)
		c.S(`* STATUS "mbox-&APYA,ADkAOsA5gD4AOU-" (MESSAGES 0)`)
		c.S(`A001 OK LIST`)
	})
}

func TestListStatusInvalid(t *testing.T) {
	runOneToOneTestWithAuth(t, defaultServerOptions(t), func(c *testConnection, _ *testSession) {
		c.C(`A001 LIST "" "*" RETURN (STATUS)`)
		c.Sx(`^A001 BAD`)

		c.C(`A002 LIST "" "*" RETURN (STATUS ())`)
		c.Sx(`^A002 BAD`)

		c.C(`A003 LIST "" "*" RETURN (STATUS (BOGUS))`)
		c.Sx(`^A003 BAD .*unknown status attribute`)

		c.C(`A004 LIST "" "*" RETURN (STATUS (MESSAGES))`)
		c.S(`* LIST (\HasNoChildren \Unmarked) "/" "INBOX"`)
		c.S(`* STATUS "INBOX" (MESSAGES 0)`)
		c.S(`A004 OK LIST`)
	})
}

func TestListExtendedPatternMatching(t *testing.T) {
	runOneToOneTestWithAuth(t, defaultServerOptions(t), func(c *testConnection, _ *testSession) {
		c.C(`A001 CREATE Work/Reports`).OK(`A001`)

		// "%" matches the top level only.
		c.C(`A002 LIST "" (%)`)
		c.S(`* LIST (\HasNoChildren \Unmarked) "/" "INBOX"`, `* LIST (\HasChildren \Unmarked) "/" "Work"`)
		c.OK(`A002`)

		// A mailbox is listed when any of the patterns matches it.
		c.C(`A003 LIST "" (INBOX Work/%)`)
		c.S(`* LIST (\HasNoChildren \Unmarked) "/" "INBOX"`, `* LIST (\HasNoChildren \Unmarked) "/" "Work/Reports"`)
		c.OK(`A003`)

		c.C(`A004 LIST "" (Drafts Sent)`)
		c.Sx(`^A004 OK`)

		// The reference is put in front of every pattern.
		c.C(`A005 LIST "Work/" (% Missing)`)
		c.S(`* LIST (\HasNoChildren \Unmarked) "/" "Work/Reports"`)
		c.OK(`A005`)

		c.C(`A006 LIST "" (inbox)`)
		c.S(`* LIST (\HasNoChildren \Unmarked) "/" "INBOX"`)
		c.OK(`A006`)
	})
}

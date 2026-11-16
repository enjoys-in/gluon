package tests

import (
	"errors"
	"testing"

	"github.com/ProtonMail/gluon/imap"
	"github.com/ProtonMail/gluon/imap/command"
)

// createSearchResMessages appends five messages to INBOX: 2, 3 and 5 are unseen and 2 is flagged.
func createSearchResMessages(c *testConnection) {
	c.doAppend("INBOX", buildRFC5322TestLiteral("To: 1@pm.me"), `\Seen`).expect("OK")
	c.doAppend("INBOX", buildRFC5322TestLiteral("To: 2@pm.me"), `\Flagged`).expect("OK")
	c.doAppend("INBOX", buildRFC5322TestLiteral("To: 3@pm.me")).expect("OK")
	c.doAppend("INBOX", buildRFC5322TestLiteral("To: 4@pm.me"), `\Seen`).expect("OK")
	c.doAppend("INBOX", buildRFC5322TestLiteral("To: 5@pm.me")).expect("OK")

	c.C(`A000 SELECT INBOX`).OK(`A000`)
}

func TestSearchResSaveSendsNoResult(t *testing.T) {
	runOneToOneTestWithAuth(t, defaultServerOptions(t), func(c *testConnection, _ *testSession) {
		createSearchResMessages(c)

		c.C(`A001 SEARCH RETURN (SAVE) UNSEEN`).Sx(`^A001 OK `)

		c.C(`A002 FETCH $ (UID)`)
		c.S(`* 2 FETCH (UID 2)`, `* 3 FETCH (UID 3)`, `* 5 FETCH (UID 5)`)
		c.OK(`A002`)

		c.C(`A003 UID FETCH $ (RFC822.SIZE)`)
		c.Sx(`^\* 2 FETCH \(RFC822\.SIZE \d+ UID 2\)`, `^\* 3 FETCH \(RFC822\.SIZE \d+ UID 3\)`, `^\* 5 FETCH \(RFC822\.SIZE \d+ UID 5\)`)
		c.OK(`A003`)
	})
}

func TestSearchResSaveWithOtherOptions(t *testing.T) {
	runOneToOneTestWithAuth(t, defaultServerOptions(t), func(c *testConnection, _ *testSession) {
		createSearchResMessages(c)

		c.C(`A001 SEARCH RETURN (SAVE COUNT) UNSEEN`).S(`* ESEARCH (TAG "A001") COUNT 3`).OK(`A001`)
		c.C(`A002 SEARCH $`).S(`* SEARCH 2 3 5`).OK(`A002`)

		c.C(`A003 SEARCH RETURN (ALL SAVE) UNSEEN`).S(`* ESEARCH (TAG "A003") ALL 2:3,5`).OK(`A003`)
		c.C(`A004 SEARCH $`).S(`* SEARCH 2 3 5`).OK(`A004`)

		// Only the lowest and the highest message are saved when nothing but MIN and MAX is asked for.
		c.C(`A005 SEARCH RETURN (SAVE MIN) UNSEEN`).S(`* ESEARCH (TAG "A005") MIN 2`).OK(`A005`)
		c.C(`A006 SEARCH $`).S(`* SEARCH 2`).OK(`A006`)

		c.C(`A007 SEARCH RETURN (MAX SAVE) UNSEEN`).S(`* ESEARCH (TAG "A007") MAX 5`).OK(`A007`)
		c.C(`A008 SEARCH $`).S(`* SEARCH 5`).OK(`A008`)

		c.C(`A009 SEARCH RETURN (SAVE MIN MAX) UNSEEN`).S(`* ESEARCH (TAG "A009") MIN 2 MAX 5`).OK(`A009`)
		c.C(`A010 SEARCH $`).S(`* SEARCH 2 5`).OK(`A010`)

		c.C(`A011 SEARCH RETURN (SAVE MIN MAX COUNT) UNSEEN`).S(`* ESEARCH (TAG "A011") MIN 2 MAX 5 COUNT 3`).OK(`A011`)
		c.C(`A012 SEARCH $`).S(`* SEARCH 2 3 5`).OK(`A012`)

		c.C(`A013 UID SEARCH RETURN (SAVE MIN MAX) FLAGGED`).S(`* ESEARCH (TAG "A013") UID MIN 2 MAX 2`).OK(`A013`)
		c.C(`A014 SEARCH $`).S(`* SEARCH 2`).OK(`A014`)
	})
}

func TestSearchResInCommands(t *testing.T) {
	runOneToOneTestWithAuth(t, defaultServerOptions(t), func(c *testConnection, _ *testSession) {
		createSearchResMessages(c)

		c.C(`A001 CREATE Archive`).OK(`A001`)
		c.C(`A002 UID SEARCH RETURN (SAVE) UNSEEN`).OK(`A002`)

		c.C(`A003 COPY $ Archive`)
		c.Sx(`^A003 OK \[COPYUID \d+ 2:3,5 1:3\]`)

		c.C(`A004 STORE $ +FLAGS.SILENT (\Seen)`).OK(`A004`)
		c.C(`A005 SEARCH UNSEEN`).S(`* SEARCH`).OK(`A005`)

		// The saved messages are still the same three, although they no longer match the search.
		c.C(`A006 SEARCH $`).S(`* SEARCH 2 3 5`).OK(`A006`)
		c.C(`A007 UID SEARCH UID $`).S(`* SEARCH 2 3 5`).OK(`A007`)
		c.C(`A008 SEARCH NOT $`).S(`* SEARCH 1 4`).OK(`A008`)
		c.C(`A009 SEARCH $ FLAGGED`).S(`* SEARCH 2`).OK(`A009`)
		c.C(`A010 SEARCH OR $ 1`).S(`* SEARCH 1 2 3 5`).OK(`A010`)
		c.C(`A011 SORT (ARRIVAL) UTF-8 $`).S(`* SORT 2 3 5`).OK(`A011`)

		c.C(`A012 UID MOVE $ Archive`)
		c.Sx(`^\* OK \[COPYUID \d+ 2:3,5 4:6\]`, `^\* 2 EXPUNGE`, `^\* 2 EXPUNGE`, `^\* 3 EXPUNGE`)
		c.OK(`A012`)

		c.C(`A013 SEARCH ALL`).S(`* SEARCH 1 2`).OK(`A013`)
		c.C(`A014 FETCH $ (UID)`).OK(`A014`)
	})
}

func TestSearchResCanReferToItself(t *testing.T) {
	runOneToOneTestWithAuth(t, defaultServerOptions(t), func(c *testConnection, _ *testSession) {
		createSearchResMessages(c)

		c.C(`A001 SEARCH RETURN (SAVE) UNSEEN`).OK(`A001`)
		c.C(`A002 SEARCH RETURN (SAVE) $ FLAGGED`).OK(`A002`)
		c.C(`A003 FETCH $ (UID)`).S(`* 2 FETCH (UID 2)`).OK(`A003`)

		c.C(`A004 SEARCH RETURN (SAVE COUNT) NOT $`).S(`* ESEARCH (TAG "A004") COUNT 4`).OK(`A004`)
		c.C(`A005 SEARCH $`).S(`* SEARCH 1 3 4 5`).OK(`A005`)
	})
}

func TestSearchResEmptyResult(t *testing.T) {
	runOneToOneTestWithAuth(t, defaultServerOptions(t), func(c *testConnection, _ *testSession) {
		createSearchResMessages(c)

		c.C(`A001 CREATE Archive`).OK(`A001`)

		// Nothing has been saved yet.
		c.C(`A002 FETCH $ (UID)`).OK(`A002`)
		c.C(`A003 UID FETCH $ (UID)`).OK(`A003`)
		c.C(`A004 STORE $ +FLAGS (\Deleted)`).OK(`A004`)
		c.C(`A005 COPY $ Archive`).OK(`A005`)
		c.C(`A006 MOVE $ Archive`).OK(`A006`)
		c.C(`A007 SEARCH $`).S(`* SEARCH`).OK(`A007`)
		c.C(`A008 UID SEARCH UID $`).S(`* SEARCH`).OK(`A008`)
		c.C(`A009 SEARCH NOT $`).S(`* SEARCH 1 2 3 4 5`).OK(`A009`)
		c.C(`A010 UID EXPUNGE $`).OK(`A010`)
		c.C(`A011 STATUS Archive (MESSAGES)`).S(`* STATUS "Archive" (MESSAGES 0)`).OK(`A011`)

		// A search that finds nothing saves an empty result.
		c.C(`A012 SEARCH RETURN (SAVE) UNSEEN`).OK(`A012`)
		c.C(`A013 SEARCH RETURN (SAVE COUNT) SUBJECT nothing-matches`).S(`* ESEARCH (TAG "A013") COUNT 0`).OK(`A013`)
		c.C(`A014 SEARCH $`).S(`* SEARCH`).OK(`A014`)
	})
}

func TestSearchResIsForgottenWhenTheMailboxChanges(t *testing.T) {
	runOneToOneTestWithAuth(t, defaultServerOptions(t), func(c *testConnection, _ *testSession) {
		createSearchResMessages(c)

		c.C(`A001 CREATE Archive`).OK(`A001`)
		c.C(`A002 SEARCH RETURN (SAVE) UNSEEN`).OK(`A002`)
		c.C(`A003 COPY $ Archive`).OK(`A003`)

		c.C(`A004 SELECT Archive`).OK(`A004`)
		c.C(`A005 SEARCH $`).S(`* SEARCH`).OK(`A005`)

		c.C(`A006 SELECT INBOX`).OK(`A006`)
		c.C(`A007 SEARCH $`).S(`* SEARCH`).OK(`A007`)

		c.C(`A008 SEARCH RETURN (SAVE) UNSEEN`).OK(`A008`)
		c.C(`A009 EXAMINE INBOX`).OK(`A009`)
		c.C(`A010 SEARCH $`).S(`* SEARCH`).OK(`A010`)
	})
}

func TestSearchResDropsExpungedMessages(t *testing.T) {
	runOneToOneTestWithAuth(t, defaultServerOptions(t), func(c *testConnection, _ *testSession) {
		createSearchResMessages(c)

		c.C(`A001 SEARCH RETURN (SAVE) UNSEEN`).OK(`A001`)
		c.C(`A002 STORE 3 +FLAGS.SILENT (\Deleted)`).OK(`A002`)
		c.C(`A003 EXPUNGE`).S(`* 3 EXPUNGE`).OK(`A003`)

		c.C(`A004 FETCH $ (UID)`)
		c.S(`* 2 FETCH (UID 2)`, `* 4 FETCH (UID 5)`)
		c.OK(`A004`)

		c.C(`A005 UID SEARCH UID $`).S(`* SEARCH 2 5`).OK(`A005`)
	})
}

func TestSearchResIsPerSession(t *testing.T) {
	runManyToOneTestWithAuth(t, defaultServerOptions(t), []int{1, 2}, func(c map[int]*testConnection, _ *testSession) {
		createSearchResMessages(c[1])
		c[2].C(`B000 SELECT INBOX`).OK(`B000`)

		c[1].C(`A001 SEARCH RETURN (SAVE) UNSEEN`).OK(`A001`)
		c[2].C(`B001 SEARCH RETURN (SAVE) SEEN`).OK(`B001`)

		c[1].C(`A002 SEARCH $`).S(`* SEARCH 2 3 5`).OK(`A002`)
		c[2].C(`B002 SEARCH $`).S(`* SEARCH 1 4`).OK(`B002`)
	})
}

func TestSearchResFailedSearchEmptiesTheResult(t *testing.T) {
	builder := &searchConnectorBuilder{}

	runOneToOneTestWithAuth(t, defaultServerOptions(t, withConnectorBuilder(builder)), func(c *testConnection, _ *testSession) {
		createSearchResMessages(c)

		c.C(`A001 SEARCH RETURN (SAVE) UNSEEN`).OK(`A001`)

		// A command that is not understood leaves the saved result alone.
		c.C(`A002 SEARCH RETURN (SAVE) NOSUCHKEY`).BAD(`A002`)
		c.C(`A003 SEARCH $`).S(`* SEARCH 2 3 5`).OK(`A003`)

		builder.conn.setAnswer(func(imap.MailboxID, []command.SearchKey) ([]imap.MessageID, bool, error) {
			return nil, false, errors.New("search index is unavailable")
		})

		// A search that fails without saving leaves it alone too.
		c.C(`A004 SEARCH SUBJECT anything`).NO(`A004`)
		c.C(`A005 SEARCH $`).S(`* SEARCH 2 3 5`).OK(`A005`)

		c.C(`A006 SEARCH RETURN (SAVE) SUBJECT anything`).NO(`A006`)
		c.C(`A007 SEARCH $`).S(`* SEARCH`).OK(`A007`)
	})
}

func TestSearchResFailures(t *testing.T) {
	runOneToOneTestWithAuth(t, defaultServerOptions(t), func(c *testConnection, _ *testSession) {
		c.C(`A001 FETCH $ (UID)`).NO(`A001`)
		c.C(`A002 SEARCH RETURN (SAVE) ALL`).NO(`A002`)

		createSearchResMessages(c)

		c.C(`A003 FETCH $,1 (UID)`).BAD(`A003`)
		c.C(`A004 FETCH 1:$ (UID)`).BAD(`A004`)
		c.C(`A005 SEARCH RETURN (SAVE)`).BAD(`A005`)
		c.C(`A006 SORT RETURN (SAVE) (DATE) UTF-8 ALL`).BAD(`A006`)
		c.C(`A007 SEARCH RETURN (SAVE SAVE) UNSEEN`).OK(`A007`)

		c.C(`A008 CAPABILITY`)
		c.Sx(`\* CAPABILITY .* SEARCHRES `)
		c.OK(`A008`)
	})
}

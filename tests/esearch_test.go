package tests

import (
	"fmt"
	"testing"
)

func appendESearchMessages(c *testConnection) {
	for i, flags := range [][]string{{`\Seen`}, nil, {`\Seen`}, nil, nil} {
		literal := buildRFC5322TestLiteral(fmt.Sprintf("To: %v@pm.me\r\nSubject: message %v", i+1, i+1))

		c.doAppend(`INBOX`, literal, flags...).expect("OK")
	}
}

func TestESearch(t *testing.T) {
	runOneToOneTestWithAuth(t, defaultServerOptions(t), func(c *testConnection, _ *testSession) {
		appendESearchMessages(c)

		c.C(`A001 SELECT INBOX`).OK(`A001`)

		c.C(`A002 SEARCH RETURN (MIN MAX COUNT ALL) UNSEEN`)
		c.S(`* ESEARCH (TAG "A002") MIN 2 MAX 5 COUNT 3 ALL 2,4:5`)
		c.OK(`A002`)

		c.C(`A003 SEARCH RETURN (COUNT) ALL`)
		c.S(`* ESEARCH (TAG "A003") COUNT 5`)
		c.OK(`A003`)

		c.C(`A004 SEARCH RETURN (MIN) SEEN`)
		c.S(`* ESEARCH (TAG "A004") MIN 1`)
		c.OK(`A004`)

		c.C(`A005 SEARCH RETURN (MAX) SEEN`)
		c.S(`* ESEARCH (TAG "A005") MAX 3`)
		c.OK(`A005`)

		c.C(`A006 SEARCH RETURN (ALL) ALL`)
		c.S(`* ESEARCH (TAG "A006") ALL 1:5`)
		c.OK(`A006`)

		c.C(`A007 search return (count min) subject "message 4"`)
		c.S(`* ESEARCH (TAG "A007") COUNT 1 MIN 4`)
		c.OK(`A007`)
	})
}

func TestESearchEmptyReturnIsAll(t *testing.T) {
	runOneToOneTestWithAuth(t, defaultServerOptions(t), func(c *testConnection, _ *testSession) {
		appendESearchMessages(c)

		c.C(`A001 SELECT INBOX`).OK(`A001`)

		c.C(`A002 SEARCH RETURN () UNSEEN`)
		c.S(`* ESEARCH (TAG "A002") ALL 2,4:5`)
		c.OK(`A002`)
	})
}

func TestESearchWithoutReturnAnswersWithSearch(t *testing.T) {
	runOneToOneTestWithAuth(t, defaultServerOptions(t), func(c *testConnection, _ *testSession) {
		appendESearchMessages(c)

		c.C(`A001 SELECT INBOX`).OK(`A001`)

		c.C(`A002 SEARCH UNSEEN`)
		c.S(`* SEARCH 2 4 5`)
		c.OK(`A002`)

		c.C(`A003 SEARCH RECENT`)
		c.Sx(`^\* SEARCH`)
		c.OK(`A003`)
	})
}

func TestESearchNoMatches(t *testing.T) {
	runOneToOneTestWithAuth(t, defaultServerOptions(t), func(c *testConnection, _ *testSession) {
		appendESearchMessages(c)

		c.C(`A001 SELECT INBOX`).OK(`A001`)

		c.C(`A002 SEARCH RETURN (MIN MAX ALL COUNT) SUBJECT nothing`)
		c.S(`* ESEARCH (TAG "A002") COUNT 0`)
		c.OK(`A002`)

		c.C(`A003 SEARCH RETURN (MIN MAX ALL) SUBJECT nothing`)
		c.S(`* ESEARCH (TAG "A003")`)
		c.OK(`A003`)

		c.C(`A004 SEARCH RETURN () SUBJECT nothing`)
		c.S(`* ESEARCH (TAG "A004")`)
		c.OK(`A004`)
	})
}

func TestESearchEmptyMailbox(t *testing.T) {
	runOneToOneTestWithAuth(t, defaultServerOptions(t), func(c *testConnection, _ *testSession) {
		c.C(`A001 SELECT INBOX`).OK(`A001`)

		c.C(`A002 SEARCH RETURN (COUNT ALL) ALL`)
		c.S(`* ESEARCH (TAG "A002") COUNT 0`)
		c.OK(`A002`)
	})
}

func TestESearchUID(t *testing.T) {
	runOneToOneTestWithAuth(t, defaultServerOptions(t), func(c *testConnection, _ *testSession) {
		appendESearchMessages(c)

		c.C(`A001 SELECT INBOX`).OK(`A001`)
		c.C(`A002 STORE 2:3 +FLAGS.SILENT (\Deleted)`).OK(`A002`)
		c.C(`A003 EXPUNGE`).OK(`A003`)

		c.C(`A004 UID SEARCH RETURN (ALL MIN MAX COUNT) ALL`)
		c.S(`* ESEARCH (TAG "A004") UID ALL 1,4:5 MIN 1 MAX 5 COUNT 3`)
		c.OK(`A004`)

		c.C(`A005 SEARCH RETURN (ALL MIN MAX COUNT) ALL`)
		c.S(`* ESEARCH (TAG "A005") ALL 1:3 MIN 1 MAX 3 COUNT 3`)
		c.OK(`A005`)

		c.C(`A006 UID SEARCH ALL`)
		c.S(`* SEARCH 1 4 5`)
		c.OK(`A006`)
	})
}

func TestESearchEchoesTheCommandTag(t *testing.T) {
	runOneToOneTestWithAuth(t, defaultServerOptions(t), func(c *testConnection, _ *testSession) {
		appendESearchMessages(c)

		c.C(`A001 SELECT INBOX`).OK(`A001`)

		c.C(`a.b-7:x SEARCH RETURN (COUNT) ALL`)
		c.S(`* ESEARCH (TAG "a.b-7:x") COUNT 5`)
		c.OK(`a.b-7:x`)
	})
}

func TestESearchWithCharset(t *testing.T) {
	runOneToOneTestWithAuth(t, defaultServerOptions(t), func(c *testConnection, _ *testSession) {
		appendESearchMessages(c)

		c.C(`A001 SELECT INBOX`).OK(`A001`)

		c.C(`A002 SEARCH RETURN (COUNT MAX) CHARSET UTF-8 SUBJECT message`)
		c.S(`* ESEARCH (TAG "A002") COUNT 5 MAX 5`)
		c.OK(`A002`)

		c.C(`A003 SEARCH RETURN (COUNT) CHARSET bogus-charset ALL`)
		c.Sx(`^A003 NO \[BADCHARSET\]`)
	})
}

func TestESearchInvalid(t *testing.T) {
	runOneToOneTestWithAuth(t, defaultServerOptions(t), func(c *testConnection, _ *testSession) {
		appendESearchMessages(c)

		c.C(`A001 SELECT INBOX`).OK(`A001`)

		c.C(`A002 SEARCH RETURN (BOGUS) ALL`)
		c.Sx(`^A002 BAD .*unknown search return option 'BOGUS'`)

		c.C(`A003 SEARCH RETURN (PARTIAL 1:2) ALL`)
		c.Sx(`^A003 BAD .*unknown search return option 'PARTIAL'`)

		c.C(`A004 SEARCH RETURN MIN ALL`)
		c.Sx(`^A004 BAD`)

		c.C(`A005 SEARCH RETURN (MIN)`)
		c.Sx(`^A005 BAD`)

		c.C(`A006 SEARCH RETURN (MIN) RETURN (MAX) ALL`)
		c.Sx(`^A006 BAD`)

		c.C(`A007 SEARCH RETURN (COUNT) ALL`)
		c.S(`* ESEARCH (TAG "A007") COUNT 5`)
		c.OK(`A007`)
	})
}

func TestESearchNeedsSelectedMailbox(t *testing.T) {
	runOneToOneTestWithAuth(t, defaultServerOptions(t), func(c *testConnection, _ *testSession) {
		c.C(`A001 SEARCH RETURN (COUNT) ALL`)
		c.Sx(`^A001 NO`)
	})
}

package tests

import (
	"testing"
	"time"

	"github.com/ProtonMail/gluon/imap/command"
	"github.com/stretchr/testify/require"
)

// createAgedMessages adds three messages received two days, two hours and a few seconds ago.
func createAgedMessages(c *testConnection, s *testSession) {
	mboxID := s.mailboxCreated("user", []string{"Aged"})

	for _, age := range []time.Duration{48 * time.Hour, 2 * time.Hour, 5 * time.Second} {
		s.messageCreated("user", mboxID, []byte(buildRFC5322TestLiteral("Subject: aged")), time.Now().Add(-age))
	}

	s.flush("user")

	c.C(`A000 SELECT Aged`).OK(`A000`)
}

func TestSearchOlderAndYounger(t *testing.T) {
	runOneToOneTestWithAuth(t, defaultServerOptions(t), func(c *testConnection, s *testSession) {
		createAgedMessages(c, s)

		c.C(`A001 SEARCH OLDER 3600`).S(`* SEARCH 1 2`).OK(`A001`)
		c.C(`A002 SEARCH YOUNGER 3600`).S(`* SEARCH 3`).OK(`A002`)
		c.C(`A003 SEARCH OLDER 86400`).S(`* SEARCH 1`).OK(`A003`)
		c.C(`A004 SEARCH YOUNGER 86400`).S(`* SEARCH 2 3`).OK(`A004`)
		c.C(`A005 SEARCH YOUNGER 1`).S(`* SEARCH`).OK(`A005`)
		c.C(`A006 SEARCH OLDER 1`).S(`* SEARCH 1 2 3`).OK(`A006`)
		c.C(`A007 SEARCH OLDER 3600 YOUNGER 86400`).S(`* SEARCH 2`).OK(`A007`)
		c.C(`A008 SEARCH NOT OLDER 3600`).S(`* SEARCH 3`).OK(`A008`)
		c.C(`A009 SEARCH OR OLDER 86400 YOUNGER 3600`).S(`* SEARCH 1 3`).OK(`A009`)
	})
}

func TestSearchOlderAndYoungerWithOtherCommands(t *testing.T) {
	runOneToOneTestWithAuth(t, defaultServerOptions(t), func(c *testConnection, s *testSession) {
		createAgedMessages(c, s)

		c.C(`A001 UID SEARCH YOUNGER 86400`).S(`* SEARCH 2 3`).OK(`A001`)
		c.C(`A002 SEARCH RETURN (MIN COUNT) OLDER 3600`).S(`* ESEARCH (TAG "A002") MIN 1 COUNT 2`).OK(`A002`)
		c.C(`A003 SORT (REVERSE ARRIVAL) UTF-8 YOUNGER 86400`).S(`* SORT 3 2`).OK(`A003`)
		c.C(`A004 THREAD ORDEREDSUBJECT UTF-8 OLDER 86400`).S(`* THREAD (1)`).OK(`A004`)
	})
}

func TestSearchOlderAndYoungerFailures(t *testing.T) {
	runOneToOneTestWithAuth(t, defaultServerOptions(t), func(c *testConnection, s *testSession) {
		createAgedMessages(c, s)

		c.C(`A001 SEARCH OLDER`).BAD(`A001`)
		c.C(`A002 SEARCH OLDER 0`).BAD(`A002`)
		c.C(`A003 SEARCH YOUNGER soon`).BAD(`A003`)
		c.C(`A004 SEARCH YOUNGER 3600 extra`).BAD(`A004`)
		c.C(`A005 SEARCH YOUNGER 3600`).S(`* SEARCH 3`).OK(`A005`)

		c.C(`A006 CAPABILITY`)
		c.Sx(`\* CAPABILITY .* WITHIN`)
		c.OK(`A006`)
	})
}

func TestSearchOlderAndYoungerAreOfferedToTheConnector(t *testing.T) {
	builder := &searchConnectorBuilder{}

	runOneToOneTestWithAuth(t, defaultServerOptions(t, withConnectorBuilder(builder)), func(c *testConnection, s *testSession) {
		createAgedMessages(c, s)

		// The dummy connector does not know these keys, so Gluon answers.
		c.C(`A001 SEARCH YOUNGER 3600`).S(`* SEARCH 3`).OK(`A001`)
		c.C(`A002 SEARCH UNSEEN OLDER 3600`).S(`* SEARCH 1 2`).OK(`A002`)

		require.Equal(t, [][]command.SearchKey{
			{&command.SearchKeyYounger{Value: 3600}},
			{&command.SearchKeyOlder{Value: 3600}},
		}, builder.conn.searches())
	})
}

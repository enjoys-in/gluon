package tests

import (
	"fmt"
	"testing"

	"github.com/ProtonMail/gluon/imap/command"
	"github.com/stretchr/testify/require"
)

func uidBatchesTestMessage(i int) ([]byte, []string) {
	return []byte(buildRFC5322TestLiteral(fmt.Sprintf("To: %v@pm.me", i))), nil
}

// createUIDBatchesMessages fills a mailbox with messages whose UIDs are 1 to count, and selects it.
func createUIDBatchesMessages(c *testConnection, s *testSession, count int) {
	mboxID := s.mailboxCreated("user", []string{"Big"})
	s.batchMessageCreated("user", mboxID, count, uidBatchesTestMessage)
	s.flush("user")

	c.C(`A000 SELECT Big`).OK(`A000`)
}

func TestUIDBatches(t *testing.T) {
	runOneToOneTestWithAuth(t, defaultServerOptions(t), func(c *testConnection, s *testSession) {
		createUIDBatchesMessages(c, s, 250)

		// Newest first. The ranges follow each other and the last one reaches down to UID 1.
		c.C(`A001 UIDBATCHES 100`).S(`* UIDBATCHES (TAG "A001") 250:151,150:51,50:1`).OK(`A001`)
		c.C(`A002 UIDBATCHES 250`).S(`* UIDBATCHES (TAG "A002") 250:1`).OK(`A002`)
		c.C(`A003 UIDBATCHES 1000`).S(`* UIDBATCHES (TAG "A003") 250:1`).OK(`A003`)
		c.C(`A004 UIDBATCHES 125`).S(`* UIDBATCHES (TAG "A004") 250:126,125:1`).OK(`A004`)
		c.C(`A005 UIDBATCHES 10`).Sx(`^\* UIDBATCHES \(TAG "A005"\) 250:241,240:231,.*,10:1\r\n`).OK(`A005`)

		c.C(`A006 uidbatches 100`).S(`* UIDBATCHES (TAG "A006") 250:151,150:51,50:1`).OK(`A006`)
	})
}

func TestUIDBatchesRange(t *testing.T) {
	runOneToOneTestWithAuth(t, defaultServerOptions(t), func(c *testConnection, s *testSession) {
		createUIDBatchesMessages(c, s, 250)

		c.C(`A001 UIDBATCHES 100 1:1`).S(`* UIDBATCHES (TAG "A001") 250:151`).OK(`A001`)
		c.C(`A002 UIDBATCHES 100 2:3`).S(`* UIDBATCHES (TAG "A002") 150:51,50:1`).OK(`A002`)
		c.C(`A003 UIDBATCHES 100 3:3`).S(`* UIDBATCHES (TAG "A003") 50:1`).OK(`A003`)
		c.C(`A004 UIDBATCHES 100 1:3`).S(`* UIDBATCHES (TAG "A004") 250:151,150:51,50:1`).OK(`A004`)

		// Batches that do not exist are left out.
		c.C(`A005 UIDBATCHES 100 2:9`).S(`* UIDBATCHES (TAG "A005") 150:51,50:1`).OK(`A005`)
		c.C(`A006 UIDBATCHES 100 4:9`).S(`* UIDBATCHES (TAG "A006")`).OK(`A006`)
	})
}

func TestUIDBatchesFollowTheUIDsOfTheMailbox(t *testing.T) {
	runOneToOneTestWithAuth(t, defaultServerOptions(t), func(c *testConnection, s *testSession) {
		createUIDBatchesMessages(c, s, 40)

		// Without UIDs 11 to 20 and 31 to 35, a batch of 10 messages covers more UIDs.
		c.C(`A001 UID STORE 11:20 +FLAGS.SILENT (\Deleted)`).OK(`A001`)
		c.C(`A002 UID STORE 31:35 +FLAGS.SILENT (\Deleted)`).OK(`A002`)
		c.C(`A003 EXPUNGE`)

		for range 10 {
			c.S(`* 11 EXPUNGE`)
		}

		for range 5 {
			c.S(`* 21 EXPUNGE`)
		}

		c.OK(`A003`)

		// 25 messages are left: UIDs 1:10, 21:30 and 36:40. The ranges still follow each other without a gap.
		c.C(`A004 UIDBATCHES 10`).S(`* UIDBATCHES (TAG "A004") 40:26,25:6,5:1`).OK(`A004`)
		c.C(`A005 UIDBATCHES 10 2:2`).S(`* UIDBATCHES (TAG "A005") 25:6`).OK(`A005`)
		c.C(`A006 UIDBATCHES 12`).S(`* UIDBATCHES (TAG "A006") 40:24,23:2,1:1`).OK(`A006`)
	})
}

func TestUIDBatchesEmptyMailbox(t *testing.T) {
	runOneToOneTestWithAuth(t, defaultServerOptions(t), func(c *testConnection, _ *testSession) {
		c.C(`A001 SELECT INBOX`).OK(`A001`)
		c.C(`A002 UIDBATCHES 10`).S(`* UIDBATCHES (TAG "A002")`).OK(`A002`)
		c.C(`A003 UIDBATCHES 10 1:5`).S(`* UIDBATCHES (TAG "A003")`).OK(`A003`)
	})
}

func TestUIDBatchesLimits(t *testing.T) {
	runOneToOneTestWithAuth(t, defaultServerOptions(t), func(c *testConnection, s *testSession) {
		// 1001 batches of 10 messages.
		createUIDBatchesMessages(c, s, 10010)

		c.C(`A001 UIDBATCHES 9`)
		c.Sx(`^A001 NO \[TOOFEW\]`)

		c.C(`A002 UIDBATCHES 10`)
		c.Sx(`^A002 NO \[TOOMANY\]`)

		c.C(`A003 UIDBATCHES 10 1:1001`)
		c.Sx(`^A003 NO \[TOOMANY\]`)

		c.C(`A004 UIDBATCHES 10 1:1000`).Sx(`^\* UIDBATCHES \(TAG "A004"\) 10010:10001,`).OK(`A004`)
		c.C(`A005 UIDBATCHES 10 2:1001`).Sx(`^\* UIDBATCHES \(TAG "A005"\) 10000:9991,.*,10:1\r\n`).OK(`A005`)
		c.C(`A006 UIDBATCHES 11`).Sx(`^\* UIDBATCHES \(TAG "A006"\) 10010:10000,9999:9989,.*,11:1\r\n`).OK(`A006`)
	})
}

func TestUIDBatchesFailures(t *testing.T) {
	runOneToOneTest(t, defaultServerOptions(t), func(c *testConnection, _ *testSession) {
		c.C(`A001 UIDBATCHES 10`).NO(`A001`)

		c.C(`A002 LOGIN user pass`).OK(`A002`)

		// A mailbox has to be selected.
		c.C(`A003 UIDBATCHES 10`).NO(`A003`)

		c.C(`A004 SELECT INBOX`).OK(`A004`)

		for i, line := range []string{
			`UIDBATCHES`,
			`UIDBATCHES 0`,
			`UIDBATCHES -5`,
			`UIDBATCHES many`,
			`UIDBATCHES 10 0:5`,
			`UIDBATCHES 10 5:0`,
			`UIDBATCHES 10 5:2`,
			`UIDBATCHES 10 3`,
			`UIDBATCHES 10 1:`,
			`UIDBATCHES 10 1:2 extra`,
			`UIDBATCHES 10 *`,
		} {
			tag := fmt.Sprintf("B%03d", i)

			c.Cf(`%v %v`, tag, line)
			c.Sx(`^` + tag + ` BAD`)
		}

		c.C(`A005 CAPABILITY`)
		c.Sx(`\* CAPABILITY .* UIDBATCHES `)
		c.OK(`A005`)
	})
}

func TestParserUIDBatches(t *testing.T) {
	cmd, err := parseCommand(`tag UIDBATCHES 100`)
	require.NoError(t, err)
	require.Equal(t, command.Command{Tag: "tag", Payload: &command.UIDBatches{BatchSize: 100}}, cmd)
	require.False(t, cmd.Payload.(*command.UIDBatches).HasRange())

	cmd, err = parseCommand(`tag uidbatches 100 2:5`)
	require.NoError(t, err)
	require.Equal(t, command.Command{Tag: "tag", Payload: &command.UIDBatches{BatchSize: 100, From: 2, To: 5}}, cmd)
	require.True(t, cmd.Payload.(*command.UIDBatches).HasRange())
	require.Equal(t, "UIDBATCHES 100 2:5", cmd.Payload.String())
}

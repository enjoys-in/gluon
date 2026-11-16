package tests

import (
	"testing"
	"time"

	"github.com/ProtonMail/gluon/imap"
	"github.com/stretchr/testify/require"
)

// Flags other than \Seen, \Flagged and the forwarded flags reach the connector through AddOrUpdateMessageFlags and
// RemoveMessageFlags, whichever STORE form sets them.
func TestStoreReportsOtherFlagsToConnector(t *testing.T) {
	runOneToOneTestWithAuth(t, defaultServerOptions(t), func(c *testConnection, s *testSession) {
		mboxID := s.mailboxCreated("user", []string{"Box"})
		messageID := s.messageCreated("user", mboxID, []byte(buildRFC5322TestLiteral("To: 1@pm.me")), time.Now())
		s.flush("user")

		c.C(`A001 SELECT Box`).OK(`A001`)

		flags := func() imap.FlagSet { return s.connectorMessageFlags("user", messageID) }

		// +FLAGS: system flags and any keyword.
		c.C(`A002 STORE 1 +FLAGS (\Answered \Draft $Important custom-tag)`).OK(`A002`)
		require.True(t, flags().ContainsAll(imap.FlagAnswered, imap.FlagDraft, "$Important", "custom-tag"), flags().ToSlice())

		// \Seen and \Flagged keep their own connector calls.
		c.C(`A003 STORE 1 +FLAGS (\Seen \Flagged)`).OK(`A003`)
		require.False(t, flags().ContainsAny(imap.FlagSeen, imap.FlagFlagged), flags().ToSlice())

		// -FLAGS removes.
		c.C(`A004 STORE 1 -FLAGS (custom-tag)`).OK(`A004`)
		require.False(t, flags().Contains("custom-tag"), flags().ToSlice())

		// \Deleted is kept per mailbox; it is still reported both ways.
		c.C(`A005 STORE 1 +FLAGS (\Deleted)`).OK(`A005`)
		require.True(t, flags().Contains(imap.FlagDeleted), flags().ToSlice())
		c.C(`A006 STORE 1 -FLAGS (\Deleted)`).OK(`A006`)
		require.False(t, flags().Contains(imap.FlagDeleted), flags().ToSlice())

		// FLAGS replaces: what is not in the new set is removed, what is new is added.
		c.C(`A007 STORE 1 FLAGS (\Seen $Pinned)`).OK(`A007`)
		require.True(t, flags().Contains("$Pinned"), flags().ToSlice())
		require.False(t, flags().ContainsAny(imap.FlagAnswered, imap.FlagDraft, "$Important"), flags().ToSlice())
	})
}

package response

import (
	"fmt"

	"github.com/ProtonMail/gluon/imap"
)

type itemMessageLimit struct {
	limit int
	uid   imap.UID
}

// ItemMessageLimit is the MESSAGELIMIT response code (RFC 9738). The UID is the lowest one the command processed;
// it is left out when it is zero.
func ItemMessageLimit(limit int, uid imap.UID) *itemMessageLimit {
	return &itemMessageLimit{limit: limit, uid: uid}
}

func (c *itemMessageLimit) String() string {
	if c.uid == 0 {
		return fmt.Sprintf("MESSAGELIMIT %d", c.limit)
	}

	return fmt.Sprintf("MESSAGELIMIT %d %d", c.limit, c.uid)
}

package response

import (
	"fmt"

	"github.com/ProtonMail/gluon/imap"
)

type itemAppendUID struct {
	uidValidity, messageUID imap.UID
}

func ItemAppendUID(uidValidity, messageUID imap.UID) *itemAppendUID {
	return &itemAppendUID{
		uidValidity: uidValidity,
		messageUID:  messageUID,
	}
}

func (c *itemAppendUID) String() string {
	return fmt.Sprintf("APPENDUID %v %v", c.uidValidity, c.messageUID)
}

type itemAppendUIDs struct {
	uidValidity imap.UID
	messageUIDs imap.SeqSet
}

// ItemAppendUIDs is the APPENDUID item for an APPEND command with several messages (RFC 3502).
func ItemAppendUIDs(uidValidity imap.UID, messageUIDs []imap.UID) *itemAppendUIDs {
	return &itemAppendUIDs{
		uidValidity: uidValidity,
		messageUIDs: imap.NewSeqSetFromUID(messageUIDs),
	}
}

func (c *itemAppendUIDs) String() string {
	return fmt.Sprintf("APPENDUID %v %v", c.uidValidity, c.messageUIDs)
}

package response

import (
	"fmt"

	"github.com/ProtonMail/gluon/imap"
)

type expunge struct {
	seq imap.SeqID
	uid imap.UID
}

func Expunge(seq imap.SeqID) *expunge {
	return &expunge{
		seq: seq,
	}
}

// ForUID sets the UID of the message, which names it once the client has enabled UIDONLY.
func (r *expunge) ForUID(uid imap.UID) *expunge {
	r.uid = uid
	return r
}

func (r *expunge) Send(s Session) error {
	if isUIDOnly(s) {
		return s.WriteResponse(r.vanishedString())
	}

	return s.WriteResponse(r.String())
}

func (r *expunge) String() string {
	return fmt.Sprintf("* %v EXPUNGE", r.seq)
}

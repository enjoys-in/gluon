package command

import (
	"github.com/ProtonMail/gluon/rfcparser"
)

// parseSearchKeyUIDBound reads UIDAFTER and UIDBEFORE (RFC 9738). They are the same as "UID <uid>+1:*" and
// "UID 1:<uid>-1", so they become the keys a client could have written itself.
func parseSearchKeyUIDBound(p *rfcparser.Parser, after bool) (SearchKey, error) {
	// search-key =/ ( "UIDAFTER" / "UIDBEFORE" ) SP nz-number
	uid, err := parseStringKeyNumber(p)
	if err != nil {
		return nil, err
	}

	if uid == 0 {
		return nil, p.MakeError("expected a non-zero UID")
	}

	switch {
	case after:
		return &SearchKeyUID{SeqSet: []SeqRange{{Begin: SeqNum(uid + 1), End: SeqNumValueAsterisk}}}, nil

	case uid == 1:
		return &SearchKeyNot{Key: &SearchKeyAll{}}, nil

	default:
		return &SearchKeyUID{SeqSet: []SeqRange{{Begin: 1, End: SeqNum(uid - 1)}}}, nil
	}
}

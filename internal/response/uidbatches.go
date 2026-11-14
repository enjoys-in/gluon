package response

import (
	"strconv"

	"github.com/ProtonMail/gluon/imap"
)

type uidBatches struct {
	tag    string
	ranges [][2]imap.UID
}

// UIDBatches is the untagged response to UIDBATCHES: * UIDBATCHES (TAG "A1") 250:151,150:51,50:1
func UIDBatches(tag string) *uidBatches {
	return &uidBatches{tag: tag}
}

func (r *uidBatches) WithRange(high, low imap.UID) *uidBatches {
	r.ranges = append(r.ranges, [2]imap.UID{high, low})

	return r
}

func (r *uidBatches) Send(s Session) error {
	return s.WriteResponse(r.String())
}

func (r *uidBatches) String() string {
	line := make([]byte, 0, 32+len(r.tag)+len(r.ranges)*22)
	line = append(line, `* UIDBATCHES (TAG "`...)
	line = append(line, r.tag...)
	line = append(line, `")`...)

	for i, batch := range r.ranges {
		if i == 0 {
			line = append(line, ' ')
		} else {
			line = append(line, ',')
		}

		line = strconv.AppendUint(line, uint64(batch[0]), 10)
		line = append(line, ':')
		line = strconv.AppendUint(line, uint64(batch[1]), 10)
	}

	return string(line)
}

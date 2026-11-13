package response

import (
	"fmt"
	"slices"

	"github.com/ProtonMail/gluon/imap"
	"github.com/bradenaw/juniper/xslices"
)

type esearch struct {
	tag   string
	uid   bool
	seqs  []uint32
	items []string
}

func ESearch(tag string, uid bool, seqs ...uint32) *esearch {
	slices.Sort(seqs)

	return &esearch{
		tag:  tag,
		uid:  uid,
		seqs: seqs,
	}
}

func (r *esearch) WithMin() *esearch {
	if len(r.seqs) > 0 {
		r.items = append(r.items, fmt.Sprintf("MIN %v", r.seqs[0]))
	}

	return r
}

func (r *esearch) WithMax() *esearch {
	if len(r.seqs) > 0 {
		r.items = append(r.items, fmt.Sprintf("MAX %v", r.seqs[len(r.seqs)-1]))
	}

	return r
}

func (r *esearch) WithAll() *esearch {
	if len(r.seqs) > 0 {
		set := imap.NewSeqSet(xslices.Map(r.seqs, func(seq uint32) imap.SeqID { return imap.SeqID(seq) }))

		r.items = append(r.items, fmt.Sprintf("ALL %v", set))
	}

	return r
}

func (r *esearch) WithCount() *esearch {
	r.items = append(r.items, fmt.Sprintf("COUNT %v", len(r.seqs)))

	return r
}

func (r *esearch) Send(s Session) error {
	return s.WriteResponse(r.String())
}

func (r *esearch) String() string {
	parts := []string{"*", "ESEARCH", fmt.Sprintf(`(TAG "%v")`, r.tag)}

	if r.uid {
		parts = append(parts, "UID")
	}

	return join(append(parts, r.items...))
}

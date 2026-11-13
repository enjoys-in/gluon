package response

import (
	"slices"
	"strconv"
)

type esearch struct {
	tag     string
	uid     bool
	seqs    []uint32
	items   []string
	ordered bool
}

func ESearch(tag string, uid bool, seqs ...uint32) *esearch {
	slices.Sort(seqs)

	return &esearch{
		tag:  tag,
		uid:  uid,
		seqs: seqs,
	}
}

// ESort is the ESEARCH response to SORT (RFC 5267). The numbers stay in sort order: MIN and MAX are the first and
// the last of them and ALL lists them one by one.
func ESort(tag string, uid bool, seqs ...uint32) *esearch {
	return &esearch{
		tag:     tag,
		uid:     uid,
		seqs:    seqs,
		ordered: true,
	}
}

func (r *esearch) WithMin() *esearch {
	if len(r.seqs) > 0 {
		r.items = append(r.items, "MIN "+strconv.FormatUint(uint64(r.seqs[0]), 10))
	}

	return r
}

func (r *esearch) WithMax() *esearch {
	if len(r.seqs) > 0 {
		r.items = append(r.items, "MAX "+strconv.FormatUint(uint64(r.seqs[len(r.seqs)-1]), 10))
	}

	return r
}

func (r *esearch) WithAll() *esearch {
	if len(r.seqs) > 0 && r.ordered {
		r.items = append(r.items, string(appendSeqList([]byte("ALL "), r.seqs)))
	} else if len(r.seqs) > 0 {
		r.items = append(r.items, string(appendSeqSet([]byte("ALL "), r.seqs)))
	}

	return r
}

func (r *esearch) WithCount() *esearch {
	r.items = append(r.items, "COUNT "+strconv.Itoa(len(r.seqs)))

	return r
}

func (r *esearch) Send(s Session) error {
	return s.WriteResponse(r.String())
}

func (r *esearch) String() string {
	parts := []string{"*", "ESEARCH", `(TAG "` + r.tag + `")`}

	if r.uid {
		parts = append(parts, "UID")
	}

	return join(append(parts, r.items...))
}

// appendSeqSet writes sorted numbers as a sequence set, with runs of consecutive numbers as ranges.
func appendSeqSet(dst []byte, sorted []uint32) []byte {
	for i := 0; i < len(sorted); {
		begin, end := sorted[i], sorted[i]

		for i++; i < len(sorted) && sorted[i] <= end+1; i++ {
			end = sorted[i]
		}

		if len(dst) > 0 && dst[len(dst)-1] != ' ' {
			dst = append(dst, ',')
		}

		dst = strconv.AppendUint(dst, uint64(begin), 10)

		if end > begin {
			dst = append(dst, ':')
			dst = strconv.AppendUint(dst, uint64(end), 10)
		}
	}

	return dst
}

// appendSeqList writes the numbers in the given order, separated by commas.
func appendSeqList(dst []byte, seqs []uint32) []byte {
	for i, seq := range seqs {
		if i > 0 {
			dst = append(dst, ',')
		}

		dst = strconv.AppendUint(dst, uint64(seq), 10)
	}

	return dst
}

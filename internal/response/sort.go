package response

import (
	"strconv"
)

type sortResponse struct {
	seqs []uint32
}

// Sort creates a SORT untagged response: * SORT num1 num2 ...
func Sort(seqs ...uint32) *sortResponse {
	return &sortResponse{
		seqs: seqs,
	}
}

func (r *sortResponse) Send(s Session) error {
	return s.WriteResponse(r.String())
}

func (r *sortResponse) String() string {
	parts := []string{"*", "SORT"}

	if len(r.seqs) > 0 {
		var seqs []string

		for _, seq := range r.seqs {
			seqs = append(seqs, strconv.Itoa(int(seq)))
		}

		parts = append(parts, join(seqs))
	}

	return join(parts)
}

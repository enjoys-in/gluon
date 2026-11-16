package response

import "fmt"

type itemSize struct {
	n int
}

func ItemSize(n int) *itemSize {
	return &itemSize{
		n: n,
	}
}

func (s *itemSize) String() string {
	return fmt.Sprintf("SIZE %v", s.n)
}

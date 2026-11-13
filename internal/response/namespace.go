package response

import (
	"fmt"
	"strconv"
)

type namespace struct {
	del string
}

func Namespace(del string) *namespace {
	return &namespace{del: del}
}

func (r *namespace) Send(s Session) error {
	return s.WriteResponse(r.String())
}

func (r *namespace) String() string {
	del := "NIL"

	if r.del != "" {
		del = strconv.Quote(r.del)
	}

	return fmt.Sprintf(`* NAMESPACE (("" %v)) NIL NIL`, del)
}

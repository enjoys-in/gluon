package response

import (
	"fmt"
	"strconv"

	"github.com/ProtonMail/gluon/imap"
	"github.com/bradenaw/juniper/xslices"
)

type list struct {
	name, del string
	att       imap.FlagSet
	childInfo []string
}

func List() *list {
	return &list{att: imap.NewFlagSet()}
}

func (r *list) WithName(name string) *list {
	r.name = name
	return r
}

func (r *list) WithDelimiter(del string) *list {
	r.del = del
	return r
}

func (r *list) WithAttributes(att imap.FlagSet) *list {
	r.att.AddFlagSetToSelf(att)
	return r
}

func (r *list) WithChildInfo(childInfo []string) *list {
	r.childInfo = childInfo
	return r
}

func (r *list) Send(s Session) error {
	return s.WriteResponse(r.String())
}

func (r *list) String() string {
	del := "NIL"

	if r.del != "" {
		del = strconv.Quote(r.del)
	}

	res := fmt.Sprintf(`* LIST (%v) %v %v`, join(r.att.ToSlice()), del, strconv.Quote(r.name))

	if len(r.childInfo) > 0 {
		res += fmt.Sprintf(` ("CHILDINFO" (%v))`, join(xslices.Map(r.childInfo, strconv.Quote)))
	}

	return res
}

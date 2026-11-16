package response

import (
	"fmt"
)

// uidOnlySession is a session whose client has enabled UIDONLY (RFC 9586): messages are then named by their UIDs
// only, in UIDFETCH responses instead of FETCH and VANISHED responses instead of EXPUNGE.
type uidOnlySession interface {
	UIDOnly() bool
}

func isUIDOnly(s Session) bool {
	session, ok := s.(uidOnlySession)

	return ok && session.UIDOnly()
}

func (r *fetch) uidFetchString() string {
	var items []string

	for _, item := range r.items {
		items = append(items, item.String())
	}

	return fmt.Sprintf(`* %v UIDFETCH (%v)`, r.uid, join(items))
}

func (r *expunge) vanishedString() string {
	return fmt.Sprintf("* VANISHED %v", r.uid)
}

type itemUIDRequired struct{}

// ItemUIDRequired is the UIDREQUIRED response code (RFC 9586).
func ItemUIDRequired() *itemUIDRequired {
	return &itemUIDRequired{}
}

func (c *itemUIDRequired) String() string {
	return "UIDREQUIRED"
}

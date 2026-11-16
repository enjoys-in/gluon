package response

import (
	"fmt"

	"github.com/ProtonMail/gluon/imap"
)

type fetch struct {
	seq   imap.SeqID
	uid   imap.UID
	items []Item
}

func Fetch(seq imap.SeqID) *fetch {
	return &fetch{
		seq: seq,
	}
}

// ForUID sets the UID of the message, which names it once the client has enabled UIDONLY.
func (r *fetch) ForUID(uid imap.UID) *fetch {
	r.uid = uid
	return r
}

func (r *fetch) WithItems(items ...Item) *fetch {
	r.items = append(r.items, items...)
	return r
}

func (r *fetch) Send(s Session) error {
	if isUIDOnly(s) {
		return s.WriteResponse(r.uidFetchString())
	}

	return s.WriteResponse(r.String())
}

func (r *fetch) String() string {
	var items []string

	for _, item := range r.items {
		items = append(items, item.String())
	}

	return fmt.Sprintf(`* %v FETCH (%v)`, r.seq, join(items))
}

func (r *fetch) canSkip(other Response) bool {
	otherExists, isExists := other.(*exists)
	if isExists && r.seq < otherExists.count {
		return true
	}

	_, isRecent := other.(*recent)
	if isRecent {
		return true
	}

	otherFetch, isFetch := other.(*fetch)
	if isFetch && otherFetch.seq != r.seq {
		return true
	}

	return false
}

func (r *fetch) mergeWith(other Response) Response {
	otherFetch, ok := other.(*fetch)
	if !ok || otherFetch.seq != r.seq {
		return nil
	}

	for _, item := range r.items {
		otherFetch.items = appendOrMergeItem(otherFetch.items, item)
	}

	return otherFetch
}

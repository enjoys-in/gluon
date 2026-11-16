package response

import "fmt"

type bad struct {
	tag   string
	err   error
	items []Item
}

func Bad(withTag ...string) *bad {
	var tag string

	if len(withTag) > 0 {
		tag = withTag[0]
	} else {
		tag = "*"
	}

	return &bad{
		tag: tag,
	}
}

func (r *bad) WithItems(items ...Item) *bad {
	r.items = append(r.items, items...)
	return r
}

func (r *bad) WithError(err error) *bad {
	r.err = err
	return r
}

func (r *bad) Send(s Session) error {
	return s.WriteResponse(r.String())
}

func (r *bad) String() string {
	parts := []string{r.tag, "BAD"}

	if len(r.items) > 0 {
		var items []string

		for _, item := range r.items {
			items = append(items, item.String())
		}

		parts = append(parts, fmt.Sprintf("[%v]", join(items)))
	}

	if r.err != nil {
		parts = append(parts, r.err.Error())
	}

	return join(parts)
}

func (r *bad) Error() string {
	return r.err.Error()
}

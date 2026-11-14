package response

type itemObjectID struct {
	name, id string
}

// ItemMailboxID is the MAILBOXID response code and STATUS item (RFC 8474).
func ItemMailboxID(id string) *itemObjectID {
	return &itemObjectID{name: "MAILBOXID", id: id}
}

// ItemEmailID is the EMAILID fetch item (RFC 8474).
func ItemEmailID(id string) *itemObjectID {
	return &itemObjectID{name: "EMAILID", id: id}
}

// ItemThreadID is the THREADID fetch item (RFC 8474). An empty ID is sent as NIL: the message has no thread.
func ItemThreadID(id string) *itemObjectID {
	return &itemObjectID{name: "THREADID", id: id}
}

func (c *itemObjectID) String() string {
	if c.id == "" {
		return c.name + " NIL"
	}

	return c.name + " (" + c.id + ")"
}

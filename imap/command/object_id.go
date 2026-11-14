package command

import (
	"fmt"

	"github.com/ProtonMail/gluon/imap"
	"github.com/ProtonMail/gluon/rfcparser"
)

// FetchAttributeEmailID asks for the object ID of the message (RFC 8474).
type FetchAttributeEmailID struct{}

func (f FetchAttributeEmailID) String() string {
	return "EMAILID"
}

// FetchAttributeThreadID asks for the object ID of the thread the message belongs to (RFC 8474).
type FetchAttributeThreadID struct{}

func (f FetchAttributeThreadID) String() string {
	return "THREADID"
}

// SearchKeyEmailID matches the message with the given object ID (RFC 8474).
type SearchKeyEmailID struct {
	Value string
}

func (s SearchKeyEmailID) String() string {
	return fmt.Sprintf("EMAILID %v", s.Value)
}

func (s SearchKeyEmailID) SanitizedString() string {
	return s.String()
}

// SearchKeyThreadID matches the messages of the thread with the given object ID (RFC 8474).
type SearchKeyThreadID struct {
	Value string
}

func (s SearchKeyThreadID) String() string {
	return fmt.Sprintf("THREADID %v", s.Value)
}

func (s SearchKeyThreadID) SanitizedString() string {
	return s.String()
}

func parseSearchKeyObjectID(p *rfcparser.Parser) (string, error) {
	// search-key =/ "EMAILID" SP objectid / "THREADID" SP objectid
	// objectid   = 1*255(ALPHA / DIGIT / "_" / "-")
	if err := p.Consume(rfcparser.TokenTypeSP, "expected space"); err != nil {
		return "", err
	}

	offset := p.CurrentToken().Offset

	value, err := p.ParseAtom()
	if err != nil {
		return "", err
	}

	if !imap.IsObjectID(value) {
		return "", p.MakeErrorAtOffset("invalid object ID", offset)
	}

	return value, nil
}

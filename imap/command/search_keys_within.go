package command

import (
	"fmt"

	"github.com/ProtonMail/gluon/rfcparser"
)

// SearchKeyOlder matches messages whose internal date is more than Value seconds in the past (RFC 5032).
type SearchKeyOlder struct {
	Value int
}

func (s SearchKeyOlder) String() string {
	return fmt.Sprintf("OLDER %v", s.Value)
}

func (s SearchKeyOlder) SanitizedString() string {
	return s.String()
}

// SearchKeyYounger matches messages whose internal date is within the last Value seconds (RFC 5032).
type SearchKeyYounger struct {
	Value int
}

func (s SearchKeyYounger) String() string {
	return fmt.Sprintf("YOUNGER %v", s.Value)
}

func (s SearchKeyYounger) SanitizedString() string {
	return s.String()
}

func parseSearchKeyInterval(p *rfcparser.Parser) (int, error) {
	// search-key =/ ( "OLDER" / "YOUNGER" ) SP nz-number
	value, err := parseStringKeyNumber(p)
	if err != nil {
		return 0, err
	}

	if value == 0 {
		return 0, p.MakeError("expected a non-zero number of seconds")
	}

	return value, nil
}

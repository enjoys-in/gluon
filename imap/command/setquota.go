package command

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/ProtonMail/gluon/imap"
	"github.com/ProtonMail/gluon/rfcparser"
)

type SetQuota struct {
	Root   string
	Limits []imap.QuotaLimit
}

func (s SetQuota) String() string {
	return fmt.Sprintf("SETQUOTA '%v' %v", s.Root, s.Limits)
}

func (s SetQuota) SanitizedString() string {
	return s.String()
}

type SetQuotaCommandParser struct{}

func (SetQuotaCommandParser) FromParser(p *rfcparser.Parser) (Payload, error) {
	// setquota          = "SETQUOTA" SP astring SP setquota-list
	// setquota-list     = "(" [setquota-resource *(SP setquota-resource)] ")"
	// setquota-resource = atom SP number64
	if err := p.Consume(rfcparser.TokenTypeSP, "expected space after command"); err != nil {
		return nil, err
	}

	root, err := p.ParseAString()
	if err != nil {
		return nil, err
	}

	if err := p.Consume(rfcparser.TokenTypeSP, "expected space after quota root"); err != nil {
		return nil, err
	}

	if err := p.Consume(rfcparser.TokenTypeLParen, "expected ( for quota limits start"); err != nil {
		return nil, err
	}

	var limits []imap.QuotaLimit

	for first := true; !p.Check(rfcparser.TokenTypeRParen); first = false {
		if !first {
			if err := p.Consume(rfcparser.TokenTypeSP, "expected space between quota limits"); err != nil {
				return nil, err
			}
		}

		name, err := p.ParseAtom()
		if err != nil {
			return nil, err
		}

		if err := p.Consume(rfcparser.TokenTypeSP, "expected space after quota resource"); err != nil {
			return nil, err
		}

		limit, err := parseQuotaLimit(p)
		if err != nil {
			return nil, err
		}

		limits = append(limits, imap.QuotaLimit{ResourceName: strings.ToUpper(name), Limit: limit})
	}

	if err := p.Consume(rfcparser.TokenTypeRParen, "expected ) for quota limits end"); err != nil {
		return nil, err
	}

	return &SetQuota{
		Root:   root.Value,
		Limits: limits,
	}, nil
}

func parseQuotaLimit(p *rfcparser.Parser) (uint64, error) {
	offset := p.CurrentToken().Offset

	var digits []byte

	for {
		if ok, err := p.Matches(rfcparser.TokenTypeDigit); err != nil {
			return 0, err
		} else if !ok {
			break
		}

		digits = append(digits, p.PreviousToken().Value)
	}

	limit, err := strconv.ParseUint(string(digits), 10, 63)
	if err != nil {
		return 0, p.MakeErrorAtOffset("expected a quota limit", offset)
	}

	return limit, nil
}

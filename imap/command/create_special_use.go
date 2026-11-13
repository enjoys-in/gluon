package command

import (
	"strings"

	"github.com/ProtonMail/gluon/rfcparser"
)

// parseCreateSpecialUse parses the special-use attributes of a CREATE command (RFC 6154).
func parseCreateSpecialUse(p *rfcparser.Parser) ([]string, error) {
	// create-params = SP "(" "USE" SP "(" [use-attr *(SP use-attr)] ")" ")"
	// use-attr      = "\" atom
	if err := p.Consume(rfcparser.TokenTypeLParen, "expected ( for create parameters start"); err != nil {
		return nil, err
	}

	offset := p.CurrentToken().Offset

	param, err := p.ParseAtom()
	if err != nil {
		return nil, err
	}

	if !strings.EqualFold(param, "USE") {
		return nil, p.MakeErrorAtOffset("unknown create parameter '"+param+"'", offset)
	}

	if err := p.Consume(rfcparser.TokenTypeSP, "expected space after USE"); err != nil {
		return nil, err
	}

	if err := p.Consume(rfcparser.TokenTypeLParen, "expected ( for special-use attributes start"); err != nil {
		return nil, err
	}

	attributes := []string{}

	for first := true; !p.Check(rfcparser.TokenTypeRParen); first = false {
		if !first {
			if err := p.Consume(rfcparser.TokenTypeSP, "expected space between special-use attributes"); err != nil {
				return nil, err
			}
		}

		if err := p.Consume(rfcparser.TokenTypeBackslash, `expected \ for special-use attribute start`); err != nil {
			return nil, err
		}

		attribute, err := p.ParseAtom()
		if err != nil {
			return nil, err
		}

		attributes = append(attributes, `\`+attribute)
	}

	if err := p.Consume(rfcparser.TokenTypeRParen, "expected ) for special-use attributes end"); err != nil {
		return nil, err
	}

	if err := p.Consume(rfcparser.TokenTypeRParen, "expected ) for create parameters end"); err != nil {
		return nil, err
	}

	return attributes, nil
}

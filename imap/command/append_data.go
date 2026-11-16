package command

import (
	"github.com/ProtonMail/gluon/rfcparser"
)

// isAppendData reports whether the message of an APPEND command comes next, and not its date.
func isAppendData(p *rfcparser.Parser) bool {
	return p.Check(rfcparser.TokenTypeLCurly) || p.Check(rfcparser.TokenTypeTilde) || p.Check(rfcparser.TokenTypeChar)
}

// parseAppendData reads the message of an APPEND command.
func parseAppendData(p *rfcparser.Parser) ([]byte, error) {
	// append-data  = literal / literal8 / utf8-literal
	// literal8     = "~{" number ["+"] "}" CRLF *OCTET  (RFC 3516)
	// utf8-literal = "UTF8" SP "(" literal8 ")"        (RFC 6855)
	if !p.Check(rfcparser.TokenTypeChar) {
		return parseLiteral8(p)
	}

	if err := p.ConsumeBytesFold('U', 'T', 'F', '8'); err != nil {
		return nil, err
	}

	if err := p.Consume(rfcparser.TokenTypeSP, "expected space after UTF8"); err != nil {
		return nil, err
	}

	if err := p.Consume(rfcparser.TokenTypeLParen, "expected ( for UTF8 message start"); err != nil {
		return nil, err
	}

	literal, err := parseLiteral8(p)
	if err != nil {
		return nil, err
	}

	return literal, p.Consume(rfcparser.TokenTypeRParen, "expected ) for UTF8 message end")
}

func parseLiteral8(p *rfcparser.Parser) ([]byte, error) {
	if _, err := p.Matches(rfcparser.TokenTypeTilde); err != nil {
		return nil, err
	}

	return p.ParseLiteral()
}

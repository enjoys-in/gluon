package command

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/ProtonMail/gluon/rfcparser"
	"github.com/bradenaw/juniper/xslices"
)

// FetchAttributeBinary asks for a body part with its content transfer encoding removed (RFC 3516). Without a part
// it stands for the whole message.
type FetchAttributeBinary struct {
	Part    []int
	Peek    bool
	Partial *BodySectionPartial
}

func (f FetchAttributeBinary) String() string {
	if f.Peek {
		return fmt.Sprintf("BINARY.PEEK[%v]", binarySection(f.Part))
	}

	return fmt.Sprintf("BINARY[%v]", binarySection(f.Part))
}

// FetchAttributeBinarySize asks for the size a body part has once its content transfer encoding is removed.
type FetchAttributeBinarySize struct {
	Part []int
}

func (f FetchAttributeBinarySize) String() string {
	return fmt.Sprintf("BINARY.SIZE[%v]", binarySection(f.Part))
}

func binarySection(part []int) string {
	return strings.Join(xslices.Map(part, strconv.Itoa), ".")
}

func handleBinaryFetchAttribute(p *rfcparser.Parser) (FetchAttribute, error) {
	// fetch-att      =/ "BINARY" [".PEEK"] section-binary [partial] / "BINARY.SIZE" section-binary
	// section-binary = "[" [section-part] "]"
	// partial        = "<" number "." nz-number ">"
	var peek, size bool

	if ok, err := p.Matches(rfcparser.TokenTypePeriod); err != nil {
		return nil, err
	} else if ok {
		kind, err := parseFetchAttributeName(p)
		if err != nil {
			return nil, err
		}

		switch kind.Value {
		case "peek":
			peek = true
		case "size":
			size = true
		default:
			return nil, p.MakeErrorAtOffset(fmt.Sprintf("unknown fetch attribute 'BINARY.%v'", kind.Value), kind.Offset)
		}
	}

	if err := p.Consume(rfcparser.TokenTypeLBracket, "expected [ for binary section start"); err != nil {
		return nil, err
	}

	var part []int

	if !p.Check(rfcparser.TokenTypeRBracket) {
		var err error

		if part, err = parseSectionPart(p); err != nil {
			return nil, err
		}

		if p.PreviousToken().TType == rfcparser.TokenTypePeriod {
			return nil, p.MakeError("expected a part number after '.'")
		}
	}

	if err := p.Consume(rfcparser.TokenTypeRBracket, "expected ] for binary section end"); err != nil {
		return nil, err
	}

	if size {
		return &FetchAttributeBinarySize{Part: part}, nil
	}

	binary := &FetchAttributeBinary{Part: part, Peek: peek}

	if ok, err := p.Matches(rfcparser.TokenTypeLess); err != nil {
		return nil, err
	} else if ok {
		offset, err := p.ParseNumber()
		if err != nil {
			return nil, err
		}

		if err := p.Consume(rfcparser.TokenTypePeriod, "expected '.' after partial start"); err != nil {
			return nil, err
		}

		count, err := ParseNZNumber(p)
		if err != nil {
			return nil, err
		}

		if err := p.Consume(rfcparser.TokenTypeGreater, "expected > for end of partial specification"); err != nil {
			return nil, err
		}

		binary.Partial = &BodySectionPartial{Offset: int64(offset), Count: int64(count)}
	}

	return binary, nil
}

// parseAppendLiteral reads the message of an APPEND command: a literal, or a binary literal "~{n}" for a message
// with binary content (RFC 3516).
func parseAppendLiteral(p *rfcparser.Parser) ([]byte, error) {
	if _, err := p.Matches(rfcparser.TokenTypeTilde); err != nil {
		return nil, err
	}

	return p.ParseLiteral()
}

// isAppendLiteral reports whether the message of an APPEND command comes next.
func isAppendLiteral(p *rfcparser.Parser) bool {
	return p.Check(rfcparser.TokenTypeLCurly) || p.Check(rfcparser.TokenTypeTilde)
}

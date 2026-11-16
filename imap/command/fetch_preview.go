package command

import (
	"strings"

	"github.com/ProtonMail/gluon/rfcparser"
)

// FetchAttributePreview asks for a short text preview of the message (RFC 8970). With Lazy the server may answer
// NIL instead of building a preview it does not have at hand.
type FetchAttributePreview struct {
	Lazy bool
}

func (f FetchAttributePreview) String() string {
	if f.Lazy {
		return "PREVIEW (LAZY)"
	}

	return "PREVIEW"
}

// parseFetchAttributeModifiers parses the modifier list that can follow a fetch attribute.
func parseFetchAttributeModifiers(p *rfcparser.Parser, attribute FetchAttribute) error {
	// fetch-att =/ "PREVIEW" [SP "(" "LAZY" ")"]
	preview, ok := attribute.(*FetchAttributePreview)
	if !ok {
		return p.MakeError("unexpected modifiers after fetch attribute")
	}

	if err := p.Consume(rfcparser.TokenTypeLParen, "expected ( for preview modifiers start"); err != nil {
		return err
	}

	offset := p.CurrentToken().Offset

	modifier, err := p.ParseAtom()
	if err != nil {
		return err
	}

	if !strings.EqualFold(modifier, "LAZY") {
		return p.MakeErrorAtOffset("unknown preview modifier '"+modifier+"'", offset)
	}

	preview.Lazy = true

	return p.Consume(rfcparser.TokenTypeRParen, "expected ) for preview modifiers end")
}

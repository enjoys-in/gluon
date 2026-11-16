package command

import (
	"fmt"

	"github.com/ProtonMail/gluon/rfcparser"
)

type Enable struct {
	Capabilities []string
}

func (e Enable) String() string {
	return fmt.Sprintf("ENABLE %v", e.Capabilities)
}

func (e Enable) SanitizedString() string {
	return e.String()
}

type EnableCommandParser struct{}

func (EnableCommandParser) FromParser(p *rfcparser.Parser) (Payload, error) {
	// enable = "ENABLE" 1*(SP capability)
	var capabilities []string

	for first := true; first || p.Check(rfcparser.TokenTypeSP); first = false {
		if err := p.Consume(rfcparser.TokenTypeSP, "expected space before capability"); err != nil {
			return nil, err
		}

		capability, err := p.ParseAtom()
		if err != nil {
			return nil, err
		}

		capabilities = append(capabilities, capability)
	}

	return &Enable{Capabilities: capabilities}, nil
}

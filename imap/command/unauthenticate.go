package command

import (
	"github.com/ProtonMail/gluon/rfcparser"
)

type Unauthenticate struct{}

func (l Unauthenticate) String() string {
	return "UNAUTHENTICATE"
}

func (l Unauthenticate) SanitizedString() string {
	return l.String()
}

type UnauthenticateCommandParser struct{}

func (UnauthenticateCommandParser) FromParser(p *rfcparser.Parser) (Payload, error) {
	return &Unauthenticate{}, nil
}

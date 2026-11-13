package command

import (
	"github.com/ProtonMail/gluon/rfcparser"
)

type Namespace struct{}

func (n Namespace) String() string {
	return "NAMESPACE"
}

func (n Namespace) SanitizedString() string {
	return n.String()
}

type NamespaceCommandParser struct{}

func (NamespaceCommandParser) FromParser(p *rfcparser.Parser) (Payload, error) {
	return &Namespace{}, nil
}

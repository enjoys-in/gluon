package command

import (
	"fmt"
	"strings"

	"github.com/ProtonMail/gluon/rfcparser"
)

// ACLMode says how SETACL changes the rights an identifier has.
type ACLMode int

const (
	ACLModeReplace ACLMode = iota
	ACLModeAdd
	ACLModeRemove
)

type SetACL struct {
	Mailbox    string
	Identifier string
	Rights     string
	Mode       ACLMode
}

func (s SetACL) String() string {
	return fmt.Sprintf("SETACL '%v' '%v' '%v'", s.Mailbox, s.Identifier, s.Rights)
}

func (s SetACL) SanitizedString() string {
	return fmt.Sprintf("SETACL '%v' '%v' '%v'", sanitizeString(s.Mailbox), sanitizeString(s.Identifier), s.Rights)
}

type DeleteACL struct {
	Mailbox    string
	Identifier string
}

func (d DeleteACL) String() string {
	return fmt.Sprintf("DELETEACL '%v' '%v'", d.Mailbox, d.Identifier)
}

func (d DeleteACL) SanitizedString() string {
	return fmt.Sprintf("DELETEACL '%v' '%v'", sanitizeString(d.Mailbox), sanitizeString(d.Identifier))
}

type GetACL struct {
	Mailbox string
}

func (g GetACL) String() string {
	return fmt.Sprintf("GETACL '%v'", g.Mailbox)
}

func (g GetACL) SanitizedString() string {
	return fmt.Sprintf("GETACL '%v'", sanitizeString(g.Mailbox))
}

type ListRights struct {
	Mailbox    string
	Identifier string
}

func (l ListRights) String() string {
	return fmt.Sprintf("LISTRIGHTS '%v' '%v'", l.Mailbox, l.Identifier)
}

func (l ListRights) SanitizedString() string {
	return fmt.Sprintf("LISTRIGHTS '%v' '%v'", sanitizeString(l.Mailbox), sanitizeString(l.Identifier))
}

type MyRights struct {
	Mailbox string
}

func (m MyRights) String() string {
	return fmt.Sprintf("MYRIGHTS '%v'", m.Mailbox)
}

func (m MyRights) SanitizedString() string {
	return fmt.Sprintf("MYRIGHTS '%v'", sanitizeString(m.Mailbox))
}

type SetACLCommandParser struct{}

func (SetACLCommandParser) FromParser(p *rfcparser.Parser) (Payload, error) {
	// setacl     = "SETACL" SP mailbox SP identifier SP mod-rights
	// mod-rights = astring ; +rights to add, -rights to remove, rights to replace
	mailbox, identifier, err := parseACLMailboxAndIdentifier(p)
	if err != nil {
		return nil, err
	}

	if err := p.Consume(rfcparser.TokenTypeSP, "expected space after identifier"); err != nil {
		return nil, err
	}

	offset := p.CurrentToken().Offset

	rights, err := p.ParseAString()
	if err != nil {
		return nil, err
	}

	setACL := &SetACL{Mailbox: mailbox, Identifier: identifier, Rights: rights.Value}

	if value, ok := strings.CutPrefix(rights.Value, "+"); ok {
		setACL.Rights, setACL.Mode = value, ACLModeAdd
	} else if value, ok := strings.CutPrefix(rights.Value, "-"); ok {
		setACL.Rights, setACL.Mode = value, ACLModeRemove
	}

	// rights = astring ; only lowercase ASCII letters and digits are allowed
	for _, right := range setACL.Rights {
		if (right < 'a' || right > 'z') && (right < '0' || right > '9') {
			return nil, p.MakeErrorAtOffset("rights are lowercase letters and digits", offset)
		}
	}

	return setACL, nil
}

type DeleteACLCommandParser struct{}

func (DeleteACLCommandParser) FromParser(p *rfcparser.Parser) (Payload, error) {
	// deleteacl = "DELETEACL" SP mailbox SP identifier
	mailbox, identifier, err := parseACLMailboxAndIdentifier(p)
	if err != nil {
		return nil, err
	}

	return &DeleteACL{Mailbox: mailbox, Identifier: identifier}, nil
}

type GetACLCommandParser struct{}

func (GetACLCommandParser) FromParser(p *rfcparser.Parser) (Payload, error) {
	// getacl = "GETACL" SP mailbox
	mailbox, err := parseACLMailbox(p)
	if err != nil {
		return nil, err
	}

	return &GetACL{Mailbox: mailbox}, nil
}

type ListRightsCommandParser struct{}

func (ListRightsCommandParser) FromParser(p *rfcparser.Parser) (Payload, error) {
	// listrights = "LISTRIGHTS" SP mailbox SP identifier
	mailbox, identifier, err := parseACLMailboxAndIdentifier(p)
	if err != nil {
		return nil, err
	}

	return &ListRights{Mailbox: mailbox, Identifier: identifier}, nil
}

type MyRightsCommandParser struct{}

func (MyRightsCommandParser) FromParser(p *rfcparser.Parser) (Payload, error) {
	// myrights = "MYRIGHTS" SP mailbox
	mailbox, err := parseACLMailbox(p)
	if err != nil {
		return nil, err
	}

	return &MyRights{Mailbox: mailbox}, nil
}

func parseACLMailbox(p *rfcparser.Parser) (string, error) {
	if err := p.Consume(rfcparser.TokenTypeSP, "expected space after command"); err != nil {
		return "", err
	}

	mailbox, err := ParseMailbox(p)
	if err != nil {
		return "", err
	}

	return mailbox.Value, nil
}

func parseACLMailboxAndIdentifier(p *rfcparser.Parser) (string, string, error) {
	// identifier = astring
	mailbox, err := parseACLMailbox(p)
	if err != nil {
		return "", "", err
	}

	if err := p.Consume(rfcparser.TokenTypeSP, "expected space after mailbox"); err != nil {
		return "", "", err
	}

	identifier, err := p.ParseAString()
	if err != nil {
		return "", "", err
	}

	return mailbox, identifier.Value, nil
}

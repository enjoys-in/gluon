package command

import (
	"fmt"
	"strings"

	"github.com/ProtonMail/gluon/rfcparser"
)

type ListSelectOptions struct {
	Subscribed     bool
	Remote         bool
	RecursiveMatch bool
	SpecialUse     bool
}

type ListReturnOptions struct {
	Subscribed bool
	Children   bool
	SpecialUse bool
	Status     []StatusAttribute
}

// ListExtended holds the RFC 5258 parts of a LIST command.
type ListExtended struct {
	Used          bool
	MoreMailboxes []string
	Select        ListSelectOptions
	Return        ListReturnOptions
}

func (l List) Patterns() []string {
	return append([]string{l.ListMailbox}, l.Extended.MoreMailboxes...)
}

func parseListSelectOptions(p *rfcparser.Parser, extended *ListExtended) error {
	// list-select-opts = "(" [list-select-opt *(SP list-select-opt)] ")" SP
	if !p.Check(rfcparser.TokenTypeLParen) {
		return nil
	}

	extended.Used = true

	offset := p.CurrentToken().Offset

	options, err := parseListOptions(p)
	if err != nil {
		return err
	}

	for _, option := range options {
		switch option.Value {
		case "SUBSCRIBED":
			extended.Select.Subscribed = true
		case "REMOTE":
			extended.Select.Remote = true
		case "RECURSIVEMATCH":
			extended.Select.RecursiveMatch = true
		case "SPECIAL-USE":
			extended.Select.SpecialUse = true
		default:
			return p.MakeErrorAtOffset(fmt.Sprintf("unknown list selection option '%v'", option.Value), option.Offset)
		}
	}

	if extended.Select.RecursiveMatch && !extended.Select.Subscribed {
		return p.MakeErrorAtOffset("RECURSIVEMATCH requires the SUBSCRIBED selection option", offset)
	}

	return p.Consume(rfcparser.TokenTypeSP, "expected space after selection options")
}

func parseListPatterns(p *rfcparser.Parser, extended *ListExtended) (rfcparser.String, error) {
	// mbox-or-pat = list-mailbox / "(" list-mailbox *(SP list-mailbox) ")"
	if !p.Check(rfcparser.TokenTypeLParen) {
		return parseListMailbox(p)
	}

	extended.Used = true

	if err := p.Consume(rfcparser.TokenTypeLParen, "expected ( for mailbox patterns start"); err != nil {
		return rfcparser.String{}, err
	}

	first, err := parseListMailbox(p)
	if err != nil {
		return rfcparser.String{}, err
	}

	for {
		if ok, err := p.Matches(rfcparser.TokenTypeSP); err != nil {
			return rfcparser.String{}, err
		} else if !ok {
			break
		}

		pattern, err := parseListMailbox(p)
		if err != nil {
			return rfcparser.String{}, err
		}

		extended.MoreMailboxes = append(extended.MoreMailboxes, pattern.Value)
	}

	if err := p.Consume(rfcparser.TokenTypeRParen, "expected ) for mailbox patterns end"); err != nil {
		return rfcparser.String{}, err
	}

	return first, nil
}

func parseListReturnOptions(p *rfcparser.Parser, extended *ListExtended) error {
	// list-return-opts = SP "RETURN" SP "(" [return-option *(SP return-option)] ")"
	if ok, err := p.Matches(rfcparser.TokenTypeSP); err != nil {
		return err
	} else if !ok {
		return nil
	}

	extended.Used = true

	offset := p.CurrentToken().Offset

	keyword, err := p.ParseAtom()
	if err != nil {
		return err
	}

	if !strings.EqualFold(keyword, "RETURN") {
		return p.MakeErrorAtOffset(fmt.Sprintf("expected RETURN, got '%v'", keyword), offset)
	}

	if err := p.Consume(rfcparser.TokenTypeSP, "expected space after RETURN"); err != nil {
		return err
	}

	options, err := parseListOptions(p)
	if err != nil {
		return err
	}

	for _, option := range options {
		switch option.Value {
		case "SUBSCRIBED":
			extended.Return.Subscribed = true
		case "CHILDREN":
			extended.Return.Children = true
		case "SPECIAL-USE":
			extended.Return.SpecialUse = true
		case "STATUS":
			extended.Return.Status = option.Status
		default:
			return p.MakeErrorAtOffset(fmt.Sprintf("unknown list return option '%v'", option.Value), option.Offset)
		}
	}

	return nil
}

type listOption struct {
	rfcparser.String
	Status []StatusAttribute
}

func parseListOptions(p *rfcparser.Parser) ([]listOption, error) {
	// "(" [option *(SP option)] ")"
	if err := p.Consume(rfcparser.TokenTypeLParen, "expected ( for options start"); err != nil {
		return nil, err
	}

	var options []listOption

	for first := true; !p.Check(rfcparser.TokenTypeRParen); first = false {
		if !first {
			if err := p.Consume(rfcparser.TokenTypeSP, "expected space between options"); err != nil {
				return nil, err
			}
		}

		offset := p.CurrentToken().Offset

		name, err := p.ParseAtom()
		if err != nil {
			return nil, err
		}

		option := listOption{String: rfcparser.String{Value: strings.ToUpper(name), Offset: offset}}

		if option.Value == "STATUS" {
			if err := p.Consume(rfcparser.TokenTypeSP, "expected space after STATUS"); err != nil {
				return nil, err
			}

			if option.Status, err = parseStatusAttributes(p); err != nil {
				return nil, err
			}
		}

		options = append(options, option)
	}

	if err := p.Consume(rfcparser.TokenTypeRParen, "expected ) for options end"); err != nil {
		return nil, err
	}

	return options, nil
}

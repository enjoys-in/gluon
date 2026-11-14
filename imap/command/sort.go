package command

import (
	"fmt"
	"slices"
	"strings"

	"github.com/ProtonMail/gluon/rfcparser"
	"github.com/bradenaw/juniper/xslices"
)

// SortKey represents a single sort criterion per RFC 5256.
type SortKey struct {
	Reverse bool
	Field   string // "ARRIVAL", "CC", "DATE", "FROM", "SIZE", "SUBJECT", "TO", "DISPLAYFROM", "DISPLAYTO"
}

func (s SortKey) String() string {
	if s.Reverse {
		return fmt.Sprintf("REVERSE %v", s.Field)
	}

	return s.Field
}

// Sort represents a parsed SORT command per RFC 5256.
type Sort struct {
	SortKeys []SortKey
	Charset  string
	Keys     []SearchKey
	Return   []SearchReturnOption
}

func (s Sort) String() string {
	criteria := xslices.Map(s.SortKeys, func(k SortKey) string { return k.String() })

	return fmt.Sprintf("SORT (%v) %v %v", strings.Join(criteria, " "), s.Charset, s.Keys)
}

func (s Sort) SanitizedString() string {
	criteria := xslices.Map(s.SortKeys, func(k SortKey) string { return k.String() })

	return fmt.Sprintf("SORT (%v) %v %v", strings.Join(criteria, " "), s.Charset, xslices.Map(s.Keys, func(v SearchKey) string {
		return v.SanitizedString()
	}))
}

type SortCommandParser struct{}

func (SortCommandParser) FromParser(p *rfcparser.Parser) (Payload, error) {
	// sort = "SORT" SP sort-criteria SP charset 1*(SP search-key)
	// sort-criteria = "(" sort-criterion *(SP sort-criterion) ")"
	// sort-criterion = ["REVERSE" SP] sort-key
	// sort-key = "ARRIVAL" / "CC" / "DATE" / "FROM" / "SIZE" / "SUBJECT" / "TO"
	// sort-key =/ "DISPLAYFROM" / "DISPLAYTO" (RFC 5957)

	if err := p.Consume(rfcparser.TokenTypeSP, "expected space after SORT"); err != nil {
		return nil, err
	}

	returnOpts, err := parseSortReturnOptions(p)
	if err != nil {
		return nil, err
	}

	// Parse opening paren for sort criteria list.
	if err := p.Consume(rfcparser.TokenTypeLParen, "expected '(' for sort criteria"); err != nil {
		return nil, err
	}

	sortKeys, err := parseSortCriteria(p)
	if err != nil {
		return nil, err
	}

	if err := p.Consume(rfcparser.TokenTypeRParen, "expected ')' for sort criteria"); err != nil {
		return nil, err
	}

	// Parse charset.
	if err := p.Consume(rfcparser.TokenTypeSP, "expected space before charset"); err != nil {
		return nil, err
	}

	charset, err := p.ParseAString()
	if err != nil {
		return nil, err
	}

	// Parse 1*(SP search-key).
	var keys []SearchKey

	for {
		if ok, err := p.Matches(rfcparser.TokenTypeSP); err != nil {
			return nil, err
		} else if !ok {
			break
		}

		key, err := parseSearchKey(p, 0)
		if err != nil {
			return nil, err
		}

		keys = append(keys, key)
	}

	if len(keys) == 0 {
		return nil, p.MakeError("at least one search key required for SORT")
	}

	return &Sort{
		SortKeys: sortKeys,
		Charset:  charset.Value,
		Keys:     keys,
		Return:   returnOpts,
	}, nil
}

func parseSortReturnOptions(p *rfcparser.Parser) ([]SearchReturnOption, error) {
	// RFC 5267 ESORT: sort =/ "SORT" SP "RETURN" SP "(" [search-return-opt *(SP search-return-opt)] ")" SP sort-criteria ...
	if p.Check(rfcparser.TokenTypeLParen) {
		return nil, nil
	}

	keyword, err := readSearchKeyword(p)
	if err != nil {
		return nil, err
	}

	if keyword.Value != "return" {
		return nil, p.MakeErrorAtOffset("expected '(' for sort criteria", keyword.Offset)
	}

	opts, err := parseSearchReturnOptions(p)
	if err != nil {
		return nil, err
	}

	if slices.Contains(opts, SearchReturnSave) {
		return nil, p.MakeErrorAtOffset("SORT cannot save its result", keyword.Offset)
	}

	return opts, p.Consume(rfcparser.TokenTypeSP, "expected space after return options")
}

func parseSortCriteria(p *rfcparser.Parser) ([]SortKey, error) {
	var keys []SortKey

	first, err := parseSortCriterion(p)
	if err != nil {
		return nil, err
	}

	keys = append(keys, first)

	for {
		if ok, err := p.Matches(rfcparser.TokenTypeSP); err != nil {
			return nil, err
		} else if !ok {
			break
		}

		// Check if this is closing paren — if so, stop.
		if p.Check(rfcparser.TokenTypeRParen) {
			break
		}

		k, err := parseSortCriterion(p)
		if err != nil {
			return nil, err
		}

		keys = append(keys, k)
	}

	return keys, nil
}

func parseSortCriterion(p *rfcparser.Parser) (SortKey, error) {
	keyword, err := readSearchKeyword(p)
	if err != nil {
		return SortKey{}, err
	}

	reverse := false

	if keyword.Value == "reverse" {
		reverse = true

		if err := p.Consume(rfcparser.TokenTypeSP, "expected space after REVERSE"); err != nil {
			return SortKey{}, err
		}

		keyword, err = readSearchKeyword(p)
		if err != nil {
			return SortKey{}, err
		}
	}

	field := strings.ToUpper(keyword.Value)

	switch field {
	case "ARRIVAL", "CC", "DATE", "FROM", "SIZE", "SUBJECT", "TO", "DISPLAYFROM", "DISPLAYTO":
		// Valid sort key.
	default:
		return SortKey{}, p.MakeErrorAtOffset(fmt.Sprintf("unknown sort key '%v'", field), keyword.Offset)
	}

	return SortKey{
		Reverse: reverse,
		Field:   field,
	}, nil
}

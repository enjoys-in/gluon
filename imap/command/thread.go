package command

import (
	"fmt"
	"strings"

	"github.com/ProtonMail/gluon/rfcparser"
	"github.com/bradenaw/juniper/xslices"
)

// Thread represents a parsed THREAD command per RFC 5256.
type Thread struct {
	Algorithm string // "REFERENCES" or "ORDEREDSUBJECT"
	Charset   string
	Keys      []SearchKey
}

func (t Thread) String() string {
	return fmt.Sprintf("THREAD %v %v %v", t.Algorithm, t.Charset, t.Keys)
}

func (t Thread) SanitizedString() string {
	return fmt.Sprintf("THREAD %v %v %v", t.Algorithm, t.Charset, xslices.Map(t.Keys, func(v SearchKey) string {
		return v.SanitizedString()
	}))
}

type ThreadCommandParser struct{}

func (ThreadCommandParser) FromParser(p *rfcparser.Parser) (Payload, error) {
	// thread = "THREAD" SP thread-alg SP charset 1*(SP search-key)
	// thread-alg = "ORDEREDSUBJECT" / "REFERENCES" / thread-alg-ext

	if err := p.Consume(rfcparser.TokenTypeSP, "expected space after THREAD"); err != nil {
		return nil, err
	}

	// Parse algorithm name.
	algKeyword, err := readSearchKeyword(p)
	if err != nil {
		return nil, err
	}

	algorithm := strings.ToUpper(algKeyword.Value)

	switch algorithm {
	case "REFERENCES", "ORDEREDSUBJECT":
		// Valid algorithm.
	default:
		return nil, p.MakeErrorAtOffset(fmt.Sprintf("unknown thread algorithm '%v'", algorithm), algKeyword.Offset)
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

		key, err := parseSearchKey(p)
		if err != nil {
			return nil, err
		}

		keys = append(keys, key)
	}

	if len(keys) == 0 {
		return nil, p.MakeError("at least one search key required for THREAD")
	}

	return &Thread{
		Algorithm: algorithm,
		Charset:   charset.Value,
		Keys:      keys,
	}, nil
}

package command

import (
	"fmt"
	"strings"

	"github.com/ProtonMail/gluon/imap"
	"github.com/ProtonMail/gluon/rfcparser"
)

// MetadataDepth says which entries below a requested one GETMETADATA returns.
type MetadataDepth int

const (
	MetadataDepthZero MetadataDepth = iota
	MetadataDepthOne
	MetadataDepthInfinity
)

type SetMetadata struct {
	Mailbox string
	Entries []imap.MetadataEntry
}

func (s SetMetadata) String() string {
	return fmt.Sprintf("SETMETADATA '%v' %v", s.Mailbox, s.Entries)
}

func (s SetMetadata) SanitizedString() string {
	return fmt.Sprintf("SETMETADATA '%v'", sanitizeString(s.Mailbox))
}

type GetMetadata struct {
	Mailbox    string
	Entries    []string
	Depth      MetadataDepth
	HasMaxSize bool
	MaxSize    int
}

func (g GetMetadata) String() string {
	return fmt.Sprintf("GETMETADATA '%v' %v", g.Mailbox, g.Entries)
}

func (g GetMetadata) SanitizedString() string {
	return fmt.Sprintf("GETMETADATA '%v' %v", sanitizeString(g.Mailbox), g.Entries)
}

type SetMetadataCommandParser struct{}

func (SetMetadataCommandParser) FromParser(p *rfcparser.Parser) (Payload, error) {
	// setmetadata  = "SETMETADATA" SP mailbox SP entry-values
	// entry-values = "(" entry-value *(SP entry-value) ")"
	// entry-value  = entry SP value
	// value        = nstring / literal8
	if err := p.Consume(rfcparser.TokenTypeSP, "expected space after command"); err != nil {
		return nil, err
	}

	mailbox, err := ParseMailbox(p)
	if err != nil {
		return nil, err
	}

	if err := p.Consume(rfcparser.TokenTypeSP, "expected space after mailbox"); err != nil {
		return nil, err
	}

	if err := p.Consume(rfcparser.TokenTypeLParen, "expected ( for entries start"); err != nil {
		return nil, err
	}

	var entries []imap.MetadataEntry

	for first := true; first || !p.Check(rfcparser.TokenTypeRParen); first = false {
		if !first {
			if err := p.Consume(rfcparser.TokenTypeSP, "expected space between entries"); err != nil {
				return nil, err
			}
		}

		name, err := parseMetadataEntry(p)
		if err != nil {
			return nil, err
		}

		if err := p.Consume(rfcparser.TokenTypeSP, "expected space after entry"); err != nil {
			return nil, err
		}

		value, err := parseMetadataValue(p)
		if err != nil {
			return nil, err
		}

		entries = append(entries, imap.MetadataEntry{Name: name, Value: value})
	}

	if err := p.Consume(rfcparser.TokenTypeRParen, "expected ) for entries end"); err != nil {
		return nil, err
	}

	return &SetMetadata{Mailbox: mailbox.Value, Entries: entries}, nil
}

type GetMetadataCommandParser struct{}

func (GetMetadataCommandParser) FromParser(p *rfcparser.Parser) (Payload, error) {
	// getmetadata = "GETMETADATA" [SP getmetadata-options] SP mailbox SP entries
	// entries     = entry / "(" entry *(SP entry) ")"
	//
	// The examples of RFC 5464 give the options after the mailbox, so they are accepted in both places.
	if err := p.Consume(rfcparser.TokenTypeSP, "expected space after command"); err != nil {
		return nil, err
	}

	getMetadata := &GetMetadata{}

	if ok, err := p.Matches(rfcparser.TokenTypeLParen); err != nil {
		return nil, err
	} else if ok {
		if err := parseGetMetadataOptions(p, getMetadata); err != nil {
			return nil, err
		}

		if err := p.Consume(rfcparser.TokenTypeSP, "expected space after options"); err != nil {
			return nil, err
		}
	}

	mailbox, err := ParseMailbox(p)
	if err != nil {
		return nil, err
	}

	getMetadata.Mailbox = mailbox.Value

	if err := p.Consume(rfcparser.TokenTypeSP, "expected space after mailbox"); err != nil {
		return nil, err
	}

	if ok, err := p.Matches(rfcparser.TokenTypeLParen); err != nil {
		return nil, err
	} else if !ok {
		entry, err := parseMetadataEntry(p)
		if err != nil {
			return nil, err
		}

		getMetadata.Entries = []string{entry}

		return getMetadata, nil
	}

	// An option starts with a letter and an entry with a slash, a quote or a literal.
	if p.Check(rfcparser.TokenTypeChar) {
		if err := parseGetMetadataOptions(p, getMetadata); err != nil {
			return nil, err
		}

		if err := p.Consume(rfcparser.TokenTypeSP, "expected space after options"); err != nil {
			return nil, err
		}

		if ok, err := p.Matches(rfcparser.TokenTypeLParen); err != nil {
			return nil, err
		} else if !ok {
			entry, err := parseMetadataEntry(p)
			if err != nil {
				return nil, err
			}

			getMetadata.Entries = []string{entry}

			return getMetadata, nil
		}
	}

	for first := true; first || !p.Check(rfcparser.TokenTypeRParen); first = false {
		if !first {
			if err := p.Consume(rfcparser.TokenTypeSP, "expected space between entries"); err != nil {
				return nil, err
			}
		}

		entry, err := parseMetadataEntry(p)
		if err != nil {
			return nil, err
		}

		getMetadata.Entries = append(getMetadata.Entries, entry)
	}

	if err := p.Consume(rfcparser.TokenTypeRParen, "expected ) for entries end"); err != nil {
		return nil, err
	}

	return getMetadata, nil
}

// parseGetMetadataOptions parses the options of GETMETADATA after their opening parenthesis.
func parseGetMetadataOptions(p *rfcparser.Parser, getMetadata *GetMetadata) error {
	// getmetadata-options = "(" getmetadata-option *(SP getmetadata-option) ")"
	// maxsize-opt         = "MAXSIZE" SP number
	// scope-opt           = "DEPTH" SP ("0" / "1" / "infinity")
	for first := true; first || !p.Check(rfcparser.TokenTypeRParen); first = false {
		if !first {
			if err := p.Consume(rfcparser.TokenTypeSP, "expected space between options"); err != nil {
				return err
			}
		}

		offset := p.CurrentToken().Offset

		option, err := p.ParseAtom()
		if err != nil {
			return err
		}

		if err := p.Consume(rfcparser.TokenTypeSP, "expected space after option"); err != nil {
			return err
		}

		switch strings.ToUpper(option) {
		case "MAXSIZE":
			if getMetadata.MaxSize, err = p.ParseNumber(); err != nil {
				return err
			}

			getMetadata.HasMaxSize = true

		case "DEPTH":
			depth, err := p.ParseAtom()
			if err != nil {
				return err
			}

			switch strings.ToLower(depth) {
			case "0":
				getMetadata.Depth = MetadataDepthZero
			case "1":
				getMetadata.Depth = MetadataDepthOne
			case "infinity":
				getMetadata.Depth = MetadataDepthInfinity
			default:
				return p.MakeErrorAtOffset("unknown depth '"+depth+"'", offset)
			}

		default:
			return p.MakeErrorAtOffset("unknown option '"+option+"'", offset)
		}
	}

	return p.Consume(rfcparser.TokenTypeRParen, "expected ) for options end")
}

func parseMetadataEntry(p *rfcparser.Parser) (string, error) {
	// entry = astring
	offset := p.CurrentToken().Offset

	entry, err := p.ParseAString()
	if err != nil {
		return "", err
	}

	// Entry names are case-insensitive.
	name := strings.ToLower(entry.Value)

	if !isMetadataEntry(name) {
		return "", p.MakeErrorAtOffset("invalid entry name", offset)
	}

	return name, nil
}

// isMetadataEntry reports whether the name is an entry name of RFC 5464, section 3.2: a path below "/private" or
// "/shared" with at least two components, or four below "vendor".
func isMetadataEntry(name string) bool {
	components := strings.Split(name, "/")

	if len(components) < 3 || components[0] != "" || (components[1] != "private" && components[1] != "shared") {
		return false
	}

	if components[2] == "vendor" && len(components) < 5 {
		return false
	}

	for _, component := range components[1:] {
		if component == "" {
			return false
		}
	}

	for i := 0; i < len(name); i++ {
		if c := name[i]; c <= 0x19 || c > 0x7f || c == '*' || c == '%' {
			return false
		}
	}

	return true
}

func parseMetadataValue(p *rfcparser.Parser) ([]byte, error) {
	// literal8 = "~{" number "}" CRLF *OCTET
	if ok, err := p.Matches(rfcparser.TokenTypeTilde); err != nil {
		return nil, err
	} else if ok {
		return p.ParseLiteral()
	}

	value, isNil, err := ParseNString(p)
	if err != nil || isNil {
		return nil, err
	}

	return []byte(value.Value), nil
}

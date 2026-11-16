package response

import (
	"strconv"
	"strings"

	"github.com/ProtonMail/gluon/imap"
)

type metadata struct {
	mailbox string
	entries []imap.MetadataEntry
}

// Metadata is the untagged response to GETMETADATA (RFC 5464): * METADATA "INBOX" (/private/comment "My comment")
func Metadata(mailbox string, entries []imap.MetadataEntry) *metadata {
	return &metadata{mailbox: mailbox, entries: entries}
}

func (r *metadata) Send(s Session) error {
	return s.WriteResponse(r.String())
}

func (r *metadata) String() string {
	values := make([]string, 0, 2*len(r.entries))

	for _, entry := range r.entries {
		values = append(values, astring(entry.Name), metadataValue(entry.Value))
	}

	return "* METADATA " + strconv.Quote(r.mailbox) + " (" + join(values) + ")"
}

// metadataValue writes the value of an entry: NIL for an entry that does not exist, a quoted string, a literal for
// text a quoted string cannot hold, or a binary literal for a value with a zero byte.
func metadataValue(value []byte) string {
	if value == nil {
		return "NIL"
	}

	binary, literal := "", false

	for _, c := range value {
		if c == 0 {
			binary, literal = "~", true

			break
		}

		if c < ' ' || c > '~' {
			literal = true
		}
	}

	if literal {
		return binary + "{" + strconv.Itoa(len(value)) + "}\r\n" + string(value)
	}

	return `"` + strings.NewReplacer(`\`, `\\`, `"`, `\"`).Replace(string(value)) + `"`
}

type itemMetadata struct {
	code string
}

// ItemMetadataLongEntries tells a client that GETMETADATA left out values larger than the size it asked for. The
// size is that of the largest value left out.
func ItemMetadataLongEntries(size int) *itemMetadata {
	return &itemMetadata{code: "LONGENTRIES " + strconv.Itoa(size)}
}

// ItemMetadataMaxSize refuses a SETMETADATA value larger than the given size.
func ItemMetadataMaxSize(size int) *itemMetadata {
	return &itemMetadata{code: "MAXSIZE " + strconv.Itoa(size)}
}

// ItemMetadataTooMany refuses a SETMETADATA that would create more annotations than the server keeps.
func ItemMetadataTooMany() *itemMetadata {
	return &itemMetadata{code: "TOOMANY"}
}

// ItemMetadataNoPrivate refuses a SETMETADATA of a private annotation on a mailbox that has none.
func ItemMetadataNoPrivate() *itemMetadata {
	return &itemMetadata{code: "NOPRIVATE"}
}

func (c *itemMetadata) String() string {
	return "METADATA " + c.code
}

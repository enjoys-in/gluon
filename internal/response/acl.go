package response

import (
	"strconv"
	"strings"

	"github.com/ProtonMail/gluon/imap"
)

type acl struct {
	mailbox string
	entries []imap.ACLEntry
}

// ACL is the untagged response to GETACL (RFC 4314): * ACL "INBOX" fred lrswipkxtea
func ACL(mailbox string, entries []imap.ACLEntry) *acl {
	return &acl{mailbox: mailbox, entries: entries}
}

func (r *acl) Send(s Session) error {
	return s.WriteResponse(r.String())
}

func (r *acl) String() string {
	parts := []string{"* ACL", strconv.Quote(r.mailbox)}

	for _, entry := range r.entries {
		parts = append(parts, astring(entry.Identifier), astring(entry.Rights))
	}

	return join(parts)
}

type listRights struct {
	mailbox    string
	identifier string
	required   string
	optional   []string
}

// ListRights is the untagged response to LISTRIGHTS (RFC 4314): the rights the identifier always has, followed by
// the rights that can be granted to it. Rights in one optional group can only be granted together.
func ListRights(mailbox, identifier, required string, optional []string) *listRights {
	return &listRights{mailbox: mailbox, identifier: identifier, required: required, optional: optional}
}

func (r *listRights) Send(s Session) error {
	return s.WriteResponse(r.String())
}

func (r *listRights) String() string {
	parts := []string{"* LISTRIGHTS", strconv.Quote(r.mailbox), astring(r.identifier), astring(r.required)}

	for _, rights := range r.optional {
		parts = append(parts, astring(rights))
	}

	return join(parts)
}

type myRights struct {
	mailbox string
	rights  string
}

// MyRights is the untagged response to MYRIGHTS (RFC 4314): * MYRIGHTS "INBOX" lrswipkxtea
func MyRights(mailbox, rights string) *myRights {
	return &myRights{mailbox: mailbox, rights: rights}
}

func (r *myRights) Send(s Session) error {
	return s.WriteResponse(r.String())
}

func (r *myRights) String() string {
	return join([]string{"* MYRIGHTS", strconv.Quote(r.mailbox), astring(r.rights)})
}

// astring writes a value as an atom when it can be one, and otherwise as a quoted string or, for text a quoted
// string cannot hold, as a literal.
func astring(value string) string {
	quoted := value == ""

	for i := 0; i < len(value); i++ {
		switch c := value[i]; {
		case c < ' ' || c > '~':
			return "{" + strconv.Itoa(len(value)) + "}\r\n" + value

		case strings.IndexByte(` (){%*"\]`, c) >= 0:
			quoted = true
		}
	}

	if !quoted {
		return value
	}

	return `"` + strings.NewReplacer(`\`, `\\`, `"`, `\"`).Replace(value) + `"`
}

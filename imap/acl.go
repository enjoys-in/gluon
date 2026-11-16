package imap

// ACLRights holds every right of RFC 4314, in the order the RFC lists them.
const ACLRights = "lrswipkxtea"

// ACLEntry is the rights an identifier has on a mailbox (RFC 4314).
type ACLEntry struct {
	Identifier string
	Rights     string
}

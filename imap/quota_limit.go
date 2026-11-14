package imap

// QuotaLimit is the limit a client asks for with SETQUOTA (RFC 9208), for one resource of a quota root.
type QuotaLimit struct {
	ResourceName string
	Limit        uint64
}

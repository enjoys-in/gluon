package imap

// MetadataEntry is an annotation of a mailbox or of the server (RFC 5464). Its name is a path in lower case such as
// "/private/comment" or "/shared/vendor/example/color". An entry with a nil value does not exist.
type MetadataEntry struct {
	Name  string
	Value []byte
}

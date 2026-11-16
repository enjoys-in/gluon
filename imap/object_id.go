package imap

// IsObjectID reports whether the ID can be sent to a client as an object ID (RFC 8474): 1 to 255 letters, digits,
// "_" and "-".
func IsObjectID(id string) bool {
	if len(id) == 0 || len(id) > 255 {
		return false
	}

	for i := 0; i < len(id); i++ {
		switch c := id[i]; {
		case c >= 'a' && c <= 'z', c >= 'A' && c <= 'Z', c >= '0' && c <= '9', c == '_', c == '-':

		default:
			return false
		}
	}

	return true
}

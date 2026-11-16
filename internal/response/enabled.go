package response

import (
	"github.com/ProtonMail/gluon/imap"
)

type enabled struct {
	caps []imap.Capability
}

// Enabled is the untagged response to ENABLE (RFC 5161). It lists the capabilities the command enabled, which can
// be none.
func Enabled(caps ...imap.Capability) *enabled {
	return &enabled{caps: caps}
}

func (r *enabled) Send(s Session) error {
	return s.WriteResponse(r.String())
}

func (r *enabled) String() string {
	parts := []string{"*", "ENABLED"}

	for _, capability := range r.caps {
		parts = append(parts, string(capability))
	}

	return join(parts)
}

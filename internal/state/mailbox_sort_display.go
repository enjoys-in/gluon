package state

import (
	"strings"

	"github.com/ProtonMail/gluon/rfc5322"
)

// extractSortDisplayName returns what DISPLAYFROM and DISPLAYTO sort by (RFC 5957): the display name of the first
// address, or its mailbox@host when it has none.
func extractSortDisplayName(value string) string {
	addresses, err := rfc5322.ParseAddressList(value)
	if err != nil || len(addresses) == 0 {
		return ""
	}

	if name := strings.TrimSpace(addresses[0].Name); name != "" {
		return strings.ToLower(name)
	}

	return strings.ToLower(addresses[0].Address)
}

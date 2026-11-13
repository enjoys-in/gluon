package session

import (
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/ProtonMail/gluon/imap"
)

var ErrUnknownSpecialUse = errors.New("unknown special-use attribute")

// validateSpecialUse ensures that the given attributes are special-use attributes of RFC 6154 and returns them as
// an imap.FlagSet.
func validateSpecialUse(attributes []string) (imap.FlagSet, error) {
	specialUse := imap.NewFlagSet()

	for _, attribute := range attributes {
		idx := slices.IndexFunc(imap.SpecialUseAttributes, func(known string) bool {
			return strings.EqualFold(known, attribute)
		})
		if idx < 0 {
			return nil, fmt.Errorf("%w %v", ErrUnknownSpecialUse, attribute)
		}

		specialUse.AddToSelf(imap.SpecialUseAttributes[idx])
	}

	return specialUse, nil
}

package connector

import (
	"context"
	"fmt"
	"math"
	"slices"
	"strings"

	"github.com/ProtonMail/gluon/imap"
)

func (conn *Dummy) SetQuotaLimits(_ context.Context, rootName string, limits []imap.QuotaLimit) (*imap.QuotaRoot, error) {
	root, ok := conn.quotaRoots[rootName]
	if !ok {
		return nil, fmt.Errorf("no such quota root")
	}

	// Every limit is checked before one is changed, so that a bad one changes nothing.
	resources := slices.Clone(root.Resources)

	for _, limit := range limits {
		idx := slices.IndexFunc(resources, func(resource imap.QuotaResource) bool {
			return strings.EqualFold(resource.ResourceName, limit.ResourceName)
		})
		if idx < 0 {
			return nil, fmt.Errorf("no such quota resource %v", limit.ResourceName)
		}

		if limit.Limit > math.MaxUint32 {
			return nil, fmt.Errorf("quota limit for %v is too large", limit.ResourceName)
		}

		resources[idx].Limit = uint32(limit.Limit)
	}

	root.Resources = resources

	return root, nil
}

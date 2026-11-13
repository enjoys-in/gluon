package response

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestESortAllKeepsSortOrder(t *testing.T) {
	assert.Equal(t, `* ESEARCH (TAG "A1") ALL 10,1,2,3,9`, ESort("A1", false, 10, 1, 2, 3, 9).WithAll().String())
}

func TestESortMinAndMaxAreTheFirstAndLastInSortOrder(t *testing.T) {
	assert.Equal(
		t,
		`* ESEARCH (TAG "A1") UID MIN 47 MAX 10 COUNT 3`,
		ESort("A1", true, 47, 2, 10).WithMin().WithMax().WithCount().String(),
	)
}

func TestESortNoMatches(t *testing.T) {
	assert.Equal(t, `* ESEARCH (TAG "A1") COUNT 0`, ESort("A1", false).WithMin().WithMax().WithAll().WithCount().String())
}

package tests

import (
	"testing"

	"github.com/ProtonMail/gluon/internal/response"

	"github.com/stretchr/testify/assert"
)

func TestResponseESortAllKeepsSortOrder(t *testing.T) {
	assert.Equal(t, `* ESEARCH (TAG "A1") ALL 10,1,2,3,9`, response.ESort("A1", false, 10, 1, 2, 3, 9).WithAll().String())
}

func TestResponseESortMinAndMaxAreTheFirstAndLastInSortOrder(t *testing.T) {
	assert.Equal(
		t,
		`* ESEARCH (TAG "A1") UID MIN 47 MAX 10 COUNT 3`,
		response.ESort("A1", true, 47, 2, 10).WithMin().WithMax().WithCount().String(),
	)
}

func TestResponseESortNoMatches(t *testing.T) {
	assert.Equal(t, `* ESEARCH (TAG "A1") COUNT 0`, response.ESort("A1", false).WithMin().WithMax().WithAll().WithCount().String())
}

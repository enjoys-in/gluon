package tests

import (
	"testing"

	"github.com/ProtonMail/gluon/internal/response"

	"github.com/stretchr/testify/assert"
)

func TestResponseESearchAll(t *testing.T) {
	assert.Equal(t, `* ESEARCH (TAG "A1") ALL 1:3,5,9:10`, response.ESearch("A1", false, 10, 1, 2, 3, 5, 9).WithAll().String())
}

func TestResponseESearchMinMaxCount(t *testing.T) {
	assert.Equal(
		t,
		`* ESEARCH (TAG "A1") MIN 2 MAX 47 COUNT 3`,
		response.ESearch("A1", false, 47, 2, 10).WithMin().WithMax().WithCount().String(),
	)
}

func TestResponseESearchKeepsRequestedOrder(t *testing.T) {
	assert.Equal(t, `* ESEARCH (TAG "A1") COUNT 2 MAX 9 MIN 4`, response.ESearch("A1", false, 4, 9).WithCount().WithMax().WithMin().String())
}

func TestResponseESearchUID(t *testing.T) {
	assert.Equal(t, `* ESEARCH (TAG "A1") UID ALL 7`, response.ESearch("A1", true, 7).WithAll().String())
}

func TestResponseESearchNoMatchesOmitsMinMaxAll(t *testing.T) {
	assert.Equal(t, `* ESEARCH (TAG "A1")`, response.ESearch("A1", false).WithMin().WithMax().WithAll().String())
}

func TestResponseESearchNoMatchesStillCounts(t *testing.T) {
	assert.Equal(t, `* ESEARCH (TAG "A1") UID COUNT 0`, response.ESearch("A1", true).WithMin().WithAll().WithCount().String())
}

func TestResponseESearchAllWritesRunsAsRanges(t *testing.T) {
	for want, numbers := range map[string][]uint32{
		"7":           {7},
		"1:3":         {1, 2, 3},
		"1,3,5":       {1, 3, 5},
		"1:2,4:6,9":   {1, 2, 4, 5, 6, 9},
		"4":           {4, 4, 4},
		"1:3,8":       {1, 1, 2, 3, 3, 8},
		"4294967295":  {4294967295},
		"10:11,20:21": {21, 10, 20, 11},
	} {
		assert.Equal(t, `* ESEARCH (TAG "A1") ALL `+want, response.ESearch("A1", false, numbers...).WithAll().String())
	}
}

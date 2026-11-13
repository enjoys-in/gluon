package response

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestESearchAll(t *testing.T) {
	assert.Equal(t, `* ESEARCH (TAG "A1") ALL 1:3,5,9:10`, ESearch("A1", false, 10, 1, 2, 3, 5, 9).WithAll().String())
}

func TestESearchMinMaxCount(t *testing.T) {
	assert.Equal(
		t,
		`* ESEARCH (TAG "A1") MIN 2 MAX 47 COUNT 3`,
		ESearch("A1", false, 47, 2, 10).WithMin().WithMax().WithCount().String(),
	)
}

func TestESearchKeepsRequestedOrder(t *testing.T) {
	assert.Equal(t, `* ESEARCH (TAG "A1") COUNT 2 MAX 9 MIN 4`, ESearch("A1", false, 4, 9).WithCount().WithMax().WithMin().String())
}

func TestESearchUID(t *testing.T) {
	assert.Equal(t, `* ESEARCH (TAG "A1") UID ALL 7`, ESearch("A1", true, 7).WithAll().String())
}

func TestESearchNoMatchesOmitsMinMaxAll(t *testing.T) {
	assert.Equal(t, `* ESEARCH (TAG "A1")`, ESearch("A1", false).WithMin().WithMax().WithAll().String())
}

func TestESearchNoMatchesStillCounts(t *testing.T) {
	assert.Equal(t, `* ESEARCH (TAG "A1") UID COUNT 0`, ESearch("A1", true).WithMin().WithAll().WithCount().String())
}

func TestAppendSeqSet(t *testing.T) {
	for want, sorted := range map[string][]uint32{
		"":            nil,
		"7":           {7},
		"1:3":         {1, 2, 3},
		"1,3,5":       {1, 3, 5},
		"1:2,4:6,9":   {1, 2, 4, 5, 6, 9},
		"4":           {4, 4, 4},
		"1:3,8":       {1, 1, 2, 3, 3, 8},
		"4294967295":  {4294967295},
		"10:11,20:21": {10, 11, 20, 21},
	} {
		assert.Equal(t, want, string(appendSeqSet(nil, sorted)))
	}
}

func TestAppendSeqSetKeepsThePrefix(t *testing.T) {
	assert.Equal(t, "ALL 1:2,5", string(appendSeqSet([]byte("ALL "), []uint32{1, 2, 5})))
}

package response

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func threadNode(num uint32, children ...*ThreadNode) *ThreadNode {
	return &ThreadNode{Num: num, Children: children}
}

func TestThread(t *testing.T) {
	// RFC 5256: * THREAD (2)(3 6 (4 23)(44 7 96))
	roots := []*ThreadNode{
		threadNode(2),
		threadNode(3, threadNode(6,
			threadNode(4, threadNode(23)),
			threadNode(44, threadNode(7, threadNode(96))),
		)),
	}

	require.Equal(t, `* THREAD (2)(3 6 (4 23)(44 7 96))`, Thread(roots).String())
}

func TestThreadEmpty(t *testing.T) {
	require.Equal(t, `* THREAD`, Thread(nil).String())
}

func TestThreadSingleMessages(t *testing.T) {
	require.Equal(t, `* THREAD (1)(2)(3)`, Thread([]*ThreadNode{threadNode(1), threadNode(2), threadNode(3)}).String())
}

func TestThreadRootWithoutMessage(t *testing.T) {
	// RFC 5256: * THREAD ((3)(5))
	require.Equal(t, `* THREAD ((3)(5))`, Thread([]*ThreadNode{threadNode(0, threadNode(3), threadNode(5))}).String())
}

func TestThreadBranchAtRoot(t *testing.T) {
	require.Equal(t, `* THREAD (1 (2)(3 4))`, Thread([]*ThreadNode{threadNode(1, threadNode(2), threadNode(3, threadNode(4)))}).String())
}

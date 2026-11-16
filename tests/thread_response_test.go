package tests

import (
	"testing"

	"github.com/ProtonMail/gluon/internal/response"

	"github.com/stretchr/testify/require"
)

func threadNode(num uint32, children ...*response.ThreadNode) *response.ThreadNode {
	return &response.ThreadNode{Num: num, Children: children}
}

func TestResponseThread(t *testing.T) {
	// RFC 5256: * THREAD (2)(3 6 (4 23)(44 7 96))
	roots := []*response.ThreadNode{
		threadNode(2),
		threadNode(3, threadNode(6,
			threadNode(4, threadNode(23)),
			threadNode(44, threadNode(7, threadNode(96))),
		)),
	}

	require.Equal(t, `* THREAD (2)(3 6 (4 23)(44 7 96))`, response.Thread(roots).String())
}

func TestResponseThreadEmpty(t *testing.T) {
	require.Equal(t, `* THREAD`, response.Thread(nil).String())
}

func TestResponseThreadSingleMessages(t *testing.T) {
	require.Equal(t, `* THREAD (1)(2)(3)`, response.Thread([]*response.ThreadNode{threadNode(1), threadNode(2), threadNode(3)}).String())
}

func TestResponseThreadRootWithoutMessage(t *testing.T) {
	// RFC 5256: * THREAD ((3)(5))
	require.Equal(t, `* THREAD ((3)(5))`, response.Thread([]*response.ThreadNode{threadNode(0, threadNode(3), threadNode(5))}).String())
}

func TestResponseThreadBranchAtRoot(t *testing.T) {
	require.Equal(t, `* THREAD (1 (2)(3 4))`, response.Thread([]*response.ThreadNode{threadNode(1, threadNode(2), threadNode(3, threadNode(4)))}).String())
}

package command

import (
	"bufio"
	"bytes"
	"io"
	"testing"

	"github.com/stretchr/testify/require"
)

func newTestInputCollector(input []byte) *InputCollector {
	return NewInputCollector(bufio.NewReader(bytes.NewReader(input)))
}

func readInput(t *testing.T, collector *InputCollector, n int) {
	t.Helper()

	_, err := io.ReadFull(collector, make([]byte, n))
	require.NoError(t, err)
}

func TestInputCollector_ResetKeepsSmallBuffer(t *testing.T) {
	collector := newTestInputCollector(bytes.Repeat([]byte("a"), 1024))

	readInput(t, collector, 1024)

	capacity := cap(collector.Bytes())

	collector.Reset()

	require.Empty(t, collector.Bytes())
	require.Equal(t, capacity, cap(collector.Bytes()))
}

func TestInputCollector_ResetReusesLargeBufferForNextLargeCommand(t *testing.T) {
	collector := newTestInputCollector(bytes.Repeat([]byte("a"), 2*maxRetainedInputSize))

	collector.Grow(2 * maxRetainedInputSize)
	readInput(t, collector, 2*maxRetainedInputSize)

	capacity := cap(collector.Bytes())

	collector.Reset()

	require.Empty(t, collector.Bytes())
	require.Equal(t, capacity, cap(collector.Bytes()))
}

func TestInputCollector_ResetReleasesLargeBufferAfterSmallCommand(t *testing.T) {
	collector := newTestInputCollector(bytes.Repeat([]byte("a"), 2*maxRetainedInputSize+16))

	collector.Grow(2 * maxRetainedInputSize)
	readInput(t, collector, 2*maxRetainedInputSize)
	collector.Reset()

	readInput(t, collector, 16)
	collector.Reset()

	require.Empty(t, collector.Bytes())
	require.Less(t, cap(collector.Bytes()), maxRetainedInputSize)
}

func TestInputCollector_GrowReservesSpaceOnce(t *testing.T) {
	collector := newTestInputCollector(bytes.Repeat([]byte("a"), 100000))

	collector.Grow(100000)

	capacity := cap(collector.Bytes())
	require.GreaterOrEqual(t, capacity, 100000)

	readInput(t, collector, 100000)

	require.Len(t, collector.Bytes(), 100000)
	require.Equal(t, capacity, cap(collector.Bytes()))
}

func TestInputCollector_DiscardDoesNotKeepTheData(t *testing.T) {
	collector := newTestInputCollector([]byte("keepDROPPEDtail"))

	readInput(t, collector, 4)
	require.NoError(t, collector.Discard(7))
	readInput(t, collector, 4)

	require.Equal(t, []byte("keeptail"), collector.Bytes())
}

func TestInputCollector_DiscardPastTheEndFails(t *testing.T) {
	collector := newTestInputCollector([]byte("short"))

	require.Error(t, collector.Discard(10))
}

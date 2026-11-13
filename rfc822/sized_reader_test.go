package rfc822

import (
	"io"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSetHeaderValueNoMemCopy_ReaderReportsItsLength(t *testing.T) {
	for _, literal := range []string{
		"To: someone@pm.me\r\nSubject: hello\r\n\r\nbody",
		"\r\nbody without header",
		"",
	} {
		reader, size, err := SetHeaderValueNoMemCopy([]byte(literal), "X-Key", "value")
		require.NoError(t, err)

		sized, ok := reader.(interface{ Len() int })
		require.True(t, ok)
		require.Equal(t, size, sized.Len())

		head := make([]byte, 4)
		n, err := io.ReadFull(reader, head)
		require.NoError(t, err)
		require.Equal(t, size-n, sized.Len())

		rest, err := io.ReadAll(reader)
		require.NoError(t, err)
		require.Zero(t, sized.Len())

		expected, err := SetHeaderValue([]byte(literal), "X-Key", "value")
		require.NoError(t, err)
		require.Equal(t, string(expected), string(head)+string(rest))
	}
}

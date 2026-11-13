package store_test

import (
	"bytes"
	"io"
	"testing"

	"github.com/ProtonMail/gluon/imap"
	"github.com/ProtonMail/gluon/store"
	"github.com/stretchr/testify/require"
)

func compressibleLiteral(size int) []byte {
	return bytes.Repeat([]byte("Subject: compressible\r\n"), size/23)
}

func TestOnDiskStore_GetAllocatesTheRecordedSize(t *testing.T) {
	store, err := store.NewOnDiskStore(t.TempDir(), []byte("pass"))
	require.NoError(t, err)

	literal := compressibleLiteral(1024 * 1024)

	id := imap.NewInternalMessageID()
	require.NoError(t, store.Set(id, bytes.NewReader(literal)))

	read, err := store.Get(id)
	require.NoError(t, err)
	require.True(t, bytes.Equal(literal, read))
	require.Less(t, cap(read), len(literal)+len(literal)/4)
}

func TestOnDiskStore_GetWithoutRecordedSize(t *testing.T) {
	store, err := store.NewOnDiskStore(t.TempDir(), []byte("pass"))
	require.NoError(t, err)

	for _, literal := range [][]byte{
		[]byte("a"),
		compressibleLiteral(1024),
		compressibleLiteral(1024 * 1024),
	} {
		id := imap.NewInternalMessageID()
		require.NoError(t, store.Set(id, io.MultiReader(bytes.NewReader(literal))))

		read, err := store.Get(id)
		require.NoError(t, err)
		require.True(t, bytes.Equal(literal, read))
	}
}

func TestOnDiskStore_GetEmptyLiteral(t *testing.T) {
	store, err := store.NewOnDiskStore(t.TempDir(), []byte("pass"))
	require.NoError(t, err)

	id := imap.NewInternalMessageID()
	require.NoError(t, store.Set(id, bytes.NewReader(nil)))

	read, err := store.Get(id)
	require.NoError(t, err)
	require.Empty(t, read)
}

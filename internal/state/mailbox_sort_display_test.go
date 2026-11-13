package state

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestExtractSortDisplayName(t *testing.T) {
	for value, want := range map[string]string{
		`Zoe Adams <zoe@example.com>`:                    "zoe adams",
		`"Adams, Zoe" <zoe@example.com>`:                 "adams, zoe",
		`<Carol@Example.com>`:                            "carol@example.com",
		`bob@example.com`:                                "bob@example.com",
		`=?UTF-8?Q?=C3=89ric?= <eric@example.com>`:       "éric",
		`Zoe <zoe@example.com>, Adam <adam@example.com>`: "zoe",
		``:                 "",
		`not an address <`: "",
	} {
		require.Equal(t, want, extractSortDisplayName(value), value)
	}
}

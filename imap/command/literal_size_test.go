package command

import (
	"testing"

	"github.com/ProtonMail/gluon/rfcparser"
	"github.com/stretchr/testify/require"
)

func TestNonSynchronizingLiteralSize(t *testing.T) {
	tests := []struct {
		name      string
		lookahead byte
		rest      string
		size      int
		ok        bool
	}{
		{name: "marker at the end", lookahead: ' ', rest: "BOGUS {5+}\r\n", size: 5, ok: true},
		{name: "several digits", lookahead: 'x', rest: " {1048576+}\r\n", size: 1048576, ok: true},
		{name: "brace is the lookahead", lookahead: '{', rest: "42+}\r\n", size: 42, ok: true},
		{name: "only the marker", lookahead: 0, rest: "{7+}\r\n", size: 7, ok: true},
		{name: "synchronizing literal", lookahead: ' ', rest: "BOGUS {5}\r\n"},
		{name: "no marker", lookahead: ' ', rest: "BOGUS junk\r\n"},
		{name: "marker not at the end", lookahead: ' ', rest: "{5+} junk\r\n"},
		{name: "no digits", lookahead: ' ', rest: "{+}\r\n"},
		{name: "no brace", lookahead: ' ', rest: "5+}\r\n"},
		{name: "wrong lookahead", lookahead: '(', rest: "5+}\r\n"},
		{name: "bare line feed", lookahead: ' ', rest: "{5+}\n"},
		{name: "empty", lookahead: ' ', rest: ""},
		{name: "ten digits is too large", lookahead: ' ', rest: "{1234567890+}\r\n", size: rfcparser.MaxLiteralSize, ok: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			size, ok := nonSynchronizingLiteralSize(tc.lookahead, []byte(tc.rest))
			require.Equal(t, tc.ok, ok)
			require.Equal(t, tc.size, size)
		})
	}
}

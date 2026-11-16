package tests

import (
	"bytes"
	"testing"

	"github.com/ProtonMail/gluon/rfcparser"

	"github.com/stretchr/testify/require"
)

func newContinuationCountingParser(t *testing.T, input string) (*rfcparser.Parser, *int) {
	t.Helper()

	continuations := new(int)

	p := rfcparser.NewParserWithLiteralContinuationCb(rfcparser.NewScanner(bytes.NewReader([]byte(input))), func(string) error {
		*continuations++
		return nil
	})

	require.NoError(t, p.Advance())

	return p, continuations
}

func TestParser_ParseLiteralSynchronizing(t *testing.T) {
	p, continuations := newContinuationCountingParser(t, "{5}\r\nhello")

	literal, err := p.ParseLiteral()
	require.NoError(t, err)
	require.Equal(t, []byte("hello"), literal)
	require.Equal(t, 1, *continuations)
}

func TestParser_ParseLiteralNonSynchronizing(t *testing.T) {
	p, continuations := newContinuationCountingParser(t, "{5+}\r\nhello")

	literal, err := p.ParseLiteral()
	require.NoError(t, err)
	require.Equal(t, []byte("hello"), literal)
	require.Zero(t, *continuations)
}

func TestParser_ParseAStringNonSynchronizingLiteral(t *testing.T) {
	p, continuations := newContinuationCountingParser(t, "{11+}\r\nhello world")

	value, err := p.ParseAString()
	require.NoError(t, err)
	require.Equal(t, "hello world", value.Value)
	require.Zero(t, *continuations)
}

func TestParser_ParseLiteralNonSynchronizingInvalid(t *testing.T) {
	for name, input := range map[string]string{
		"plus before the number": "{+5}\r\nhello",
		"two plus signs":         "{5++}\r\nhello",
		"zero size":              "{0+}\r\n",
		"too large":              "{73400320+}\r\n",
		"missing line break":     "{5+}hello",
		"truncated data":         "{5+}\r\nhi",
	} {
		t.Run(name, func(t *testing.T) {
			p, _ := newContinuationCountingParser(t, input)

			_, err := p.ParseLiteral()
			require.Error(t, err)
		})
	}
}

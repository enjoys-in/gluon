package tests

import (
	"bytes"
	"fmt"
	"testing"
)

func benchmarkAppendLiteral(size int) []byte {
	header := "From: sender@pm.me\r\nTo: user@pm.me\r\nSubject: large\r\nDate: Mon, 7 Feb 1994 21:52:25 -0800\r\n\r\n"
	line := bytes.Repeat([]byte("a"), 74)

	literal := make([]byte, 0, size)
	literal = append(literal, header...)

	for len(literal)+len(line)+2 <= size {
		literal = append(literal, line...)
		literal = append(literal, "\r\n"...)
	}

	return literal
}

func BenchmarkAppendLarge(b *testing.B) {
	runOneToOneTestWithAuth(b, defaultServerOptions(b), func(c *testConnection, _ *testSession) {
		literal := benchmarkAppendLiteral(4 * 1024 * 1024)

		commands := make([][]byte, b.N)
		for i := range commands {
			commands[i] = fmt.Appendf(make([]byte, 0, len(literal)+64), "B%v APPEND INBOX {%v+}\r\n%s", i, len(literal), literal)
		}

		b.ReportAllocs()
		b.SetBytes(int64(len(literal)))
		b.ResetTimer()

		for i := range b.N {
			c.Cb(commands[i]).OK(fmt.Sprintf("B%v", i))
		}
	})
}

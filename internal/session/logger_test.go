package session

import (
	"bytes"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func escapedLogLine(line string) string {
	line = strings.ReplaceAll(strings.ReplaceAll(strings.ReplaceAll(strings.TrimSpace(line), "\r", `\r`), "\n", `\n`), "\t", `\t`)

	if len(line) > maxLineLength {
		line = line[:maxLineLength] + "..."
	}

	return fmt.Sprintf("C[1]: %v\n", line)
}

func TestWriteLog(t *testing.T) {
	long := strings.Repeat("a", maxLineLength)

	for _, line := range []string{
		"",
		"   ",
		"A001 NOOP\r\n",
		"  A001 LOGIN user pass\t\r\n",
		long,
		long + "b",
		long + "\r\n",
		long + "\r\nb",
		long[:maxLineLength-1] + "\r\n",
		long[:maxLineLength-1] + "\r\nb",
		long[:maxLineLength-1] + " \r\n b",
		strings.Repeat("\r\n", maxLineLength) + "a",
		"A001 APPEND INBOX {4096+}\r\n" + strings.Repeat("line\r\n", 1024),
	} {
		var asString, asBytes bytes.Buffer

		writeLog(&asString, "C", "1", line)
		writeLogBytes(&asBytes, "C", "1", []byte(line))

		require.Equal(t, escapedLogLine(line), asString.String())
		require.Equal(t, escapedLogLine(line), asBytes.String())
	}
}

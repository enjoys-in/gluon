package session

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"strings"
)

func writeLog(w io.Writer, leader, sessionID, line string) {
	writeTrimmedLog(w, leader, sessionID, strings.TrimSpace(line))
}

func writeLogBytes(w io.Writer, leader, sessionID string, line []byte) {
	line = bytes.TrimSpace(line)

	if len(line) > maxLineLength {
		line = line[:maxLineLength+1]
	}

	writeTrimmedLog(w, leader, sessionID, string(line))
}

func writeTrimmedLog(w io.Writer, leader, sessionID, line string) {
	truncated := len(line) > maxLineLength
	if truncated {
		line = line[:maxLineLength]
	}

	line = strings.ReplaceAll(strings.ReplaceAll(strings.ReplaceAll(line, "\r", `\r`), "\n", `\n`), "\t", `\t`)

	if truncated || len(line) > maxLineLength {
		line = line[:maxLineLength] + "..."
	}

	if _, err := fmt.Fprintf(w, "%v[%v]: %v\n", leader, sessionID, line); err != nil {
		log.Printf("gluon: failed to write log: %v", err)
	}
}

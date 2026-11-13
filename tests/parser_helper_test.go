package tests

import (
	"strings"

	"github.com/ProtonMail/gluon/imap/command"
	"github.com/ProtonMail/gluon/rfcparser"
)

// newCommandParser returns a command parser that reads the given lines. A line that follows a literal marker is
// the data of that literal.
func newCommandParser(lines []string, options ...command.Option) *command.Parser {
	input := strings.Join(lines, "\r\n") + "\r\n"

	return command.NewParser(rfcparser.NewScanner(strings.NewReader(input)), options...)
}

// parseCommand parses one command without a server.
func parseCommand(lines ...string) (command.Command, error) {
	return newCommandParser(lines).Parse()
}

// countContinuations counts the continuation requests the parser sends while it reads literals.
func countContinuations(count *int) command.Option {
	return command.WithLiteralContinuationCallback(func(string) error {
		*count++

		return nil
	})
}

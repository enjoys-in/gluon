package tests

import (
	"fmt"
	"io"
	"regexp"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
)

type imapLoggerOption struct {
	in, out io.Writer
}

func (o *imapLoggerOption) apply(options *serverOptions) {
	options.loggerIn = o.in
	options.loggerOut = o.out
}

// withIMAPLogger makes the server write what it reads and what it sends to the given writers.
func withIMAPLogger(in, out io.Writer) serverOption {
	return &imapLoggerOption{in: in, out: out}
}

// logLines collects the lines written to it.
type logLines struct {
	lock  sync.Mutex
	lines []string
}

func (l *logLines) Write(p []byte) (int, error) {
	l.lock.Lock()
	defer l.lock.Unlock()

	l.lines = append(l.lines, strings.TrimSuffix(string(p), "\n"))

	return len(p), nil
}

// find returns the logged lines that match, without the leader that names the session.
func (l *logLines) find(pattern string) []string {
	l.lock.Lock()
	defer l.lock.Unlock()

	re := regexp.MustCompile(`^[CS]\[\d+\]: (` + pattern + `)$`)

	var found []string

	for _, line := range l.lines {
		if match := re.FindStringSubmatch(line); match != nil {
			found = append(found, match[1])
		}
	}

	return found
}

// The server cuts a logged line after this many bytes.
const maxLoggedLineLength = 120

func TestLoggerWritesEachCommandAndResponseOnOneLine(t *testing.T) {
	in, out := &logLines{}, &logLines{}

	runOneToOneTestWithAuth(t, defaultServerOptions(t, withIMAPLogger(in, out)), func(c *testConnection, _ *testSession) {
		c.C("A001 NOOP").OK("A001")
		c.C("A002 BOGUS").BAD("A002")

		require.Equal(t, []string{"A001 NOOP"}, in.find(`A001 .*`))
		require.Len(t, out.find(`A001 OK .*`), 1)
		require.Len(t, out.find(`A002 BAD .*`), 1)
	})
}

func TestLoggerEscapesLineBreaksInsideACommand(t *testing.T) {
	in, out := &logLines{}, &logLines{}

	runOneToOneTestWithAuth(t, defaultServerOptions(t, withIMAPLogger(in, out)), func(c *testConnection, _ *testSession) {
		c.C("A001 CREATE {4+}\r\nWork").OK("A001")

		require.Equal(t, []string{`A001 CREATE {4+}\r\nWork`}, in.find(`A001 .*`))
	})
}

func TestLoggerCutsLongLines(t *testing.T) {
	in, out := &logLines{}, &logLines{}

	runOneToOneTestWithAuth(t, defaultServerOptions(t, withIMAPLogger(in, out)), func(c *testConnection, _ *testSession) {
		literal := buildRFC5322TestLiteral("To: 1@pm.me") + strings.Repeat("a line of the body\r\n", 4096)
		command := fmt.Sprintf("A001 APPEND INBOX {%v+}\r\n%v", len(literal), literal)

		c.C(command).OK("A001")

		// The line is escaped first and then cut, so that the log never holds more than the limit.
		escaped := strings.NewReplacer("\r", `\r`, "\n", `\n`, "\t", `\t`).Replace(command)

		require.Equal(t, []string{escaped[:maxLoggedLineLength] + "..."}, in.find(`A001 .*`))

		// A line of exactly the limit is not cut.
		exact := "A002 CREATE " + strings.Repeat("b", maxLoggedLineLength-len("A002 CREATE "))

		c.C(exact).OK("A002")
		c.C(exact + "c").OK("A002")

		require.Equal(t, []string{exact, exact + "..."}, in.find(`A002 .*`))
	})
}

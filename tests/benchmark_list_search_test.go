package tests

import (
	"fmt"
	"testing"
)

func benchmarkMessage(n int) ([]byte, []string) {
	flags := []string{}
	if n%2 == 0 {
		flags = append(flags, `\Seen`)
	}

	return fmt.Appendf(nil, "To: %v@pm.me\r\nSubject: message %v\r\n\r\nbody", n, n), flags
}

func benchmarkCommand(b *testing.B, c *testConnection, command string) {
	b.ReportAllocs()
	b.ResetTimer()

	for i := range b.N {
		tag := fmt.Sprintf("B%v", i)
		c.Cf(`%v %v`, tag, command).OK(tag)
	}
}

func BenchmarkListHierarchy(b *testing.B) {
	runOneToOneTestWithAuth(b, defaultServerOptions(b), func(c *testConnection, s *testSession) {
		for i := range 100 {
			s.mailboxCreated("user", []string{fmt.Sprintf("top%v", i), "mid", "leaf"})
		}

		s.flush("user")

		for _, command := range []string{`LIST "" "*"`, `LIST (SUBSCRIBED RECURSIVEMATCH) "" "%" RETURN (CHILDREN)`} {
			b.Run(command, func(b *testing.B) {
				benchmarkCommand(b, c, command)
			})
		}
	})
}

func BenchmarkListStatus(b *testing.B) {
	runOneToOneTestWithAuth(b, defaultServerOptions(b), func(c *testConnection, s *testSession) {
		for i := range 20 {
			mboxID := s.mailboxCreated("user", []string{fmt.Sprintf("box%v", i)})
			s.batchMessageCreated("user", mboxID, 2000, benchmarkMessage)
		}

		s.flush("user")

		benchmarkCommand(b, c, `LIST "" "*" RETURN (STATUS (MESSAGES UNSEEN RECENT UIDNEXT UIDVALIDITY))`)
	})
}

func BenchmarkESearch(b *testing.B) {
	runOneToOneTestWithAuth(b, defaultServerOptions(b), func(c *testConnection, s *testSession) {
		mboxID := s.mailboxCreated("user", []string{"big"})
		s.batchMessageCreated("user", mboxID, 20000, benchmarkMessage)
		s.flush("user")

		c.C(`A0 SELECT big`).OK(`A0`)

		for _, command := range []string{`SEARCH RETURN (COUNT) ALL`, `SEARCH RETURN (ALL) UNSEEN`} {
			b.Run(command, func(b *testing.B) {
				benchmarkCommand(b, c, command)
			})
		}
	})
}

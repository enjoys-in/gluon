package session

import (
	"context"
	"errors"
	"strings"
	"unicode/utf8"

	"github.com/ProtonMail/gluon/imap"
	"github.com/ProtonMail/gluon/imap/command"
	"github.com/ProtonMail/gluon/internal/response"
	"github.com/emersion/go-imap/utf7"
)

var (
	ErrInvalidUTF8MailboxName = errors.New("invalid UTF-8 mailbox name")
	ErrCharsetWithUTF8        = errors.New("a charset cannot be given once UTF8=ACCEPT is enabled")
)

// handleEnable enables the capabilities that have to be asked for (RFC 5161). UTF8=ACCEPT is the only one. As the
// RFC asks, other names are ignored and the response lists what this command enabled.
func (s *Session) handleEnable(_ context.Context, tag string, cmd *command.Enable, ch chan response.Response) error {
	var enabled []imap.Capability

	for _, capability := range cmd.Capabilities {
		if strings.EqualFold(capability, string(imap.UTF8Accept)) && !s.utf8Enabled {
			s.utf8Enabled = true

			enabled = append(enabled, imap.UTF8Accept)
		}
	}

	ch <- response.Enabled(enabled...)

	ch <- response.Ok(tag).WithMessage("ENABLE")

	return nil
}

// mailboxNameToUTF8 converts a mailbox name from how the client writes it: from modified UTF-7, or from UTF-8 once
// the client has enabled UTF8=ACCEPT (RFC 6855).
func (s *Session) mailboxNameToUTF8(name string) (string, error) {
	if !s.utf8Enabled {
		return utf7.Encoding.NewDecoder().String(name)
	}

	if !utf8.ValidString(name) {
		return "", ErrInvalidUTF8MailboxName
	}

	// RFC 6855 does not allow control characters, line separators and paragraph separators in a mailbox name.
	for _, r := range name {
		if r <= 0x1f || (r >= 0x7f && r <= 0x9f) || r == 0x2028 || r == 0x2029 {
			return "", ErrInvalidUTF8MailboxName
		}
	}

	return name, nil
}

// mailboxNameFromUTF8 converts a mailbox name to how the client reads it: modified UTF-7, or UTF-8 once the client
// has enabled UTF8=ACCEPT.
func (s *Session) mailboxNameFromUTF8(name string) (string, error) {
	if s.utf8Enabled {
		return name, nil
	}

	return utf7.Encoding.NewEncoder().String(name)
}

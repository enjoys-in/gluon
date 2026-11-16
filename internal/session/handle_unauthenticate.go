package session

import (
	"context"
	"errors"
	"slices"

	"github.com/ProtonMail/gluon/imap"
	"github.com/ProtonMail/gluon/imap/command"
	"github.com/ProtonMail/gluon/internal/response"
)

var ErrUnauthenticateDisabled = errors.New("UNAUTHENTICATE is disabled")

// DisableUnauthenticate makes the session refuse the UNAUTHENTICATE command. RFC 8437 asks for a way to switch it
// off where it is not needed.
func (s *Session) DisableUnauthenticate() {
	s.unauthenticateDisabled = true

	s.remCapability(imap.UNAUTHENTICATE)
}

// handleUnauthenticate never answers NO, which RFC 8437 does not permit: the command either succeeds or, where it
// is not valid, is answered with BAD.
func (s *Session) handleUnauthenticate(ctx context.Context, tag string, _ *command.Unauthenticate, ch chan response.Response) error {
	s.userLock.Lock()
	defer s.userLock.Unlock()

	s.capsLock.Lock()
	defer s.capsLock.Unlock()

	if s.unauthenticateDisabled {
		return response.Bad(tag).WithError(ErrUnauthenticateDisabled)
	}

	if s.state == nil {
		return response.Bad(tag).WithError(ErrNotAuthenticated)
	}

	if err := s.state.ReleaseState(ctx); err != nil {
		return err
	}

	s.state = nil

	// What the client enabled with ENABLE ends with the login.
	s.utf8Enabled = false
	s.uidOnly.Store(false)

	caps := slices.DeleteFunc(slices.Clone(s.caps), func(c imap.Capability) bool {
		return !imap.IsCapabilityAvailableBeforeAuth(c)
	})

	ch <- response.Ok(tag).WithItems(response.ItemCapability(s.advertised(caps)...)).WithMessage("UNAUTHENTICATE")

	return nil
}

package session

import (
	"context"
	"slices"

	"github.com/ProtonMail/gluon/imap"
	"github.com/ProtonMail/gluon/imap/command"
	"github.com/ProtonMail/gluon/internal/response"
)

func (s *Session) handleUnauthenticate(ctx context.Context, tag string, _ *command.Unauthenticate, ch chan response.Response) error {
	s.capsLock.Lock()
	defer s.capsLock.Unlock()

	if err := s.state.ReleaseState(ctx); err != nil {
		return err
	}

	s.state = nil

	caps := slices.DeleteFunc(slices.Clone(s.caps), func(c imap.Capability) bool {
		return !imap.IsCapabilityAvailableBeforeAuth(c)
	})

	ch <- response.Ok(tag).WithItems(response.ItemCapability(s.advertised(caps)...)).WithMessage("UNAUTHENTICATE")

	return nil
}

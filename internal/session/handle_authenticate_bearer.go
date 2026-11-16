package session

import (
	"context"

	"github.com/ProtonMail/gluon/imap"
	"github.com/ProtonMail/gluon/imap/command"
	"github.com/ProtonMail/gluon/internal/response"
	"github.com/ProtonMail/gluon/internal/state"
	"github.com/ProtonMail/gluon/profiling"
)

// SetBearerAuthentication lets clients log in with an OAuth 2.0 bearer token.
func (s *Session) SetBearerAuthentication() {
	if s.disableIMAPAuthenticate {
		return
	}

	s.bearerAuthentication = true

	s.addCapability(imap.AUTHOAuthBearer)
	s.addCapability(imap.AUTHXOAuth2)
}

// handleAuthenticateBearer handles AUTHENTICATE with the OAUTHBEARER or XOAUTH2 mechanism. A token that is refused
// is answered with NO right away, without the error challenge the mechanisms can send first.
func (s *Session) handleAuthenticateBearer(ctx context.Context, tag string, cmd *command.AuthenticateBearer, ch chan response.Response) error {
	profiling.Start(ctx, profiling.CmdTypeLogin)
	defer profiling.Stop(ctx, profiling.CmdTypeLogin)

	return s.login(tag, cmd.UserID, ch, func() (*state.State, error) {
		return s.backend.GetStateWithToken(ctx, cmd.UserID, []byte(cmd.Token), s.sessionID)
	})
}

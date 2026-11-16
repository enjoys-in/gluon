package backend

import (
	"context"

	"github.com/ProtonMail/gluon/connector"
	"github.com/ProtonMail/gluon/internal/state"
)

// GetStateWithToken is GetState for a client that logs in with an OAuth 2.0 bearer token instead of a password.
// The username is empty when the client leaves it to the token to name the user.
func (b *Backend) GetStateWithToken(ctx context.Context, username string, token []byte, _ int) (*state.State, error) {
	b.usersLock.Lock()
	defer b.usersLock.Unlock()

	userID, err := b.authorizeUser(func(conn connector.Connector) bool {
		return conn.AuthorizeToken(ctx, username, token)
	})
	if err != nil {
		return nil, err
	}

	return b.newUserState(userID, username)
}

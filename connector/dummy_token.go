package connector

import (
	"bytes"
	"context"
	"slices"
)

// AuthorizeToken accepts the token set with SetBearerToken, for one of the connector's usernames or for a client
// that names no user.
func (conn *Dummy) AuthorizeToken(_ context.Context, username string, token []byte) bool {
	if len(conn.bearerToken) == 0 || !bytes.Equal(token, conn.bearerToken) {
		return false
	}

	return username == "" || slices.Contains(conn.usernames, username)
}

// SetBearerToken sets the OAuth 2.0 bearer token the connector accepts. Without one it accepts none.
func (conn *Dummy) SetBearerToken(token string) {
	conn.bearerToken = []byte(token)
}

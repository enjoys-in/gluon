package command

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"strings"

	"github.com/ProtonMail/gluon/rfcparser"
)

// AuthenticateBearer is an AUTHENTICATE command with an OAuth 2.0 bearer token: the OAUTHBEARER mechanism
// (RFC 7628) or XOAUTH2. UserID is empty when the client leaves it to the token to name the user.
type AuthenticateBearer struct {
	Mechanism string
	UserID    string
	Token     string
}

func (l AuthenticateBearer) String() string {
	return fmt.Sprintf("AUTHENTICATE %v '%v'", l.Mechanism, l.UserID)
}

func (l AuthenticateBearer) SanitizedString() string {
	return "AUTHENTICATE <AUTH_DATA>"
}

type withBearerAuthentication struct{}

func (opt withBearerAuthentication) config(builder *parserBuilder) {
	builder.bearerAuthentication = true
}

// WithBearerAuthentication makes AUTHENTICATE accept the OAUTHBEARER and XOAUTH2 mechanisms.
func WithBearerAuthentication() Option {
	return &withBearerAuthentication{}
}

const (
	mechanismOAuthBearer = "OAUTHBEARER"
	mechanismXOAuth2     = "XOAUTH2"
)

func isBearerMechanism(mechanism string) bool {
	return strings.EqualFold(mechanism, mechanismOAuthBearer) || strings.EqualFold(mechanism, mechanismXOAuth2)
}

func parseAuthBearer(p *rfcparser.Parser, mechanism string) (*AuthenticateBearer, error) {
	var (
		parsed rfcparser.String
		err    error
	)

	// RFC 4959 SASL-IR: the response can follow the mechanism.
	if ok, matchErr := p.Matches(rfcparser.TokenTypeSP); matchErr != nil {
		return nil, matchErr
	} else if ok {
		parsed, err = p.ParseAString()
	} else {
		parsed, err = p.ParseStringAfterContinuation("")
	}

	if err != nil {
		return nil, err
	}

	if parsed.Value == "*" && p.Check(rfcparser.TokenTypeCR) {
		return nil, p.MakeError(messageClientAbortedAuthentication)
	}

	decoded, err := base64.StdEncoding.DecodeString(parsed.Value)
	if err != nil {
		return nil, p.MakeError(messageInvalidBase64Content)
	}

	auth := &AuthenticateBearer{Mechanism: strings.ToUpper(mechanism)}

	// Both mechanisms send fields that end with a 0x01 byte.
	fields := bytes.Split(decoded, []byte{1})

	if auth.Mechanism == mechanismOAuthBearer {
		// OAUTHBEARER starts with a GS2 header that can name the user: "n,a=user@example.com,".
		header := strings.Split(string(fields[0]), ",")
		if len(header) != 3 || header[2] != "" {
			return nil, p.MakeError(messageInvalidAuthenticationData)
		}

		if user, ok := strings.CutPrefix(header[1], "a="); ok {
			auth.UserID = strings.NewReplacer("=2C", ",", "=3D", "=").Replace(user)
		} else if header[1] != "" {
			return nil, p.MakeError(messageInvalidAuthenticationData)
		}

		fields = fields[1:]
	}

	for _, field := range fields {
		key, value, _ := strings.Cut(string(field), "=")

		switch {
		case strings.EqualFold(key, "auth"):
			scheme, token, _ := strings.Cut(value, " ")
			if !strings.EqualFold(scheme, "Bearer") {
				return nil, p.MakeError(messageInvalidAuthenticationData)
			}

			auth.Token = strings.TrimSpace(token)

		case key == "user" && auth.Mechanism == mechanismXOAuth2:
			auth.UserID = value
		}
	}

	if auth.Token == "" {
		return nil, p.MakeError(messageInvalidAuthenticationData)
	}

	return auth, nil
}

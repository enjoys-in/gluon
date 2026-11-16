package tests

import (
	"encoding/base64"
	"testing"

	"github.com/ProtonMail/gluon/imap/command"
	"github.com/stretchr/testify/require"
)

type oauthLoginOption struct{}

func (oauthLoginOption) apply(options *serverOptions) {
	options.oauthLogin = true
}

// withOAuthLogin lets clients log in with an OAuth 2.0 bearer token.
func withOAuthLogin() serverOption {
	return oauthLoginOption{}
}

const testBearerToken = "ya29.vF9dft4qmTc2Nvb3RlckBhdHRhdmlzdGEuY29tCg"

// oauthBearer builds the response of the OAUTHBEARER mechanism (RFC 7628).
func oauthBearer(user, token string) string {
	header := "n,,"
	if user != "" {
		header = "n,a=" + user + ","
	}

	return base64.StdEncoding.EncodeToString([]byte(header + "\x01host=localhost\x01port=143\x01auth=Bearer " + token + "\x01\x01"))
}

// xoauth2 builds the response of the XOAUTH2 mechanism.
func xoauth2(user, token string) string {
	return base64.StdEncoding.EncodeToString([]byte("user=" + user + "\x01auth=Bearer " + token + "\x01\x01"))
}

func runOAuthLoginTest(t *testing.T, tests func(c *testConnection, s *testSession)) {
	runOneToOneTest(t, defaultServerOptions(t, withOAuthLogin()), func(c *testConnection, s *testSession) {
		s.conns[s.userIDs["user"]].SetBearerToken(testBearerToken)

		tests(c, s)
	})
}

func TestOAuthLoginWithOAuthBearer(t *testing.T) {
	runOAuthLoginTest(t, func(c *testConnection, _ *testSession) {
		c.C("A001 CAPABILITY")
		c.S(`* CAPABILITY AUTH=OAUTHBEARER AUTH=PLAIN AUTH=XOAUTH2 ID IDLE IMAP4rev1 LITERAL+ SASL-IR STARTTLS`)
		c.OK("A001")

		c.C("A002 AUTHENTICATE OAUTHBEARER " + oauthBearer("user", testBearerToken))
		c.Sx(`^A002 OK \[CAPABILITY .*AUTH=OAUTHBEARER.*UIDPLUS.*\] Logged in`)

		c.C("A003 SELECT INBOX").OK("A003")
	})
}

func TestOAuthLoginWithXOAuth2(t *testing.T) {
	runOAuthLoginTest(t, func(c *testConnection, _ *testSession) {
		c.C("A001 authenticate xoauth2 " + xoauth2("user", testBearerToken)).OK("A001")
		c.C("A002 SELECT INBOX").OK("A002")
	})
}

func TestOAuthLoginAfterAContinuation(t *testing.T) {
	runOAuthLoginTest(t, func(c *testConnection, _ *testSession) {
		c.C("A001 AUTHENTICATE OAUTHBEARER")
		c.S("+")
		c.C(oauthBearer("user", testBearerToken)).OK("A001")

		c.C("A002 UNAUTHENTICATE").OK("A002")

		c.C("A003 AUTHENTICATE XOAUTH2")
		c.S("+")
		c.C(xoauth2("user", testBearerToken)).OK("A003")
	})
}

func TestOAuthLoginWithoutAUserName(t *testing.T) {
	runOAuthLoginTest(t, func(c *testConnection, _ *testSession) {
		// The token alone names the user.
		c.C("A001 AUTHENTICATE OAUTHBEARER " + oauthBearer("", testBearerToken)).OK("A001")
		c.C("A002 SELECT INBOX").OK("A002")
	})
}

func TestOAuthLoginRefused(t *testing.T) {
	runOAuthLoginTest(t, func(c *testConnection, _ *testSession) {
		c.C("A001 AUTHENTICATE OAUTHBEARER " + oauthBearer("user", "wrong-token")).NO("A001")
		c.C("A002 AUTHENTICATE XOAUTH2 " + xoauth2("user", "wrong-token")).NO("A002")
		c.C("A003 AUTHENTICATE OAUTHBEARER " + oauthBearer("someone-else", testBearerToken)).NO("A003")
		c.C("A004 SELECT INBOX").NO("A004")

		// The password is not a token and the token is not a password.
		c.C("A005 AUTHENTICATE OAUTHBEARER " + oauthBearer("user", "pass")).NO("A005")
		c.C("A006 LOGIN user " + testBearerToken).NO("A006")

		c.C("A007 AUTHENTICATE XOAUTH2 " + xoauth2("user", testBearerToken)).OK("A007")
		c.C("A008 AUTHENTICATE XOAUTH2 " + xoauth2("user", testBearerToken)).BAD("A008")
	})
}

func TestOAuthLoginInvalidResponse(t *testing.T) {
	b64 := func(s string) string { return base64.StdEncoding.EncodeToString([]byte(s)) }

	runOAuthLoginTest(t, func(c *testConnection, _ *testSession) {
		c.C("A001 AUTHENTICATE OAUTHBEARER not-base64").BAD("A001")
		c.C("A002 AUTHENTICATE OAUTHBEARER " + b64("n,a=user,\x01host=localhost\x01\x01")).BAD("A002")
		c.C("A003 AUTHENTICATE OAUTHBEARER " + b64("n,a=user,\x01auth=Basic dXNlcjpwYXNz\x01\x01")).BAD("A003")
		c.C("A004 AUTHENTICATE OAUTHBEARER " + b64("n,a=user,\x01auth=Bearer \x01\x01")).BAD("A004")
		c.C("A005 AUTHENTICATE OAUTHBEARER " + b64("user=user\x01auth=Bearer "+testBearerToken+"\x01\x01")).BAD("A005")
		c.C("A006 AUTHENTICATE OAUTHBEARER " + b64("n,x=user,\x01auth=Bearer "+testBearerToken+"\x01\x01")).BAD("A006")
		c.C("A007 AUTHENTICATE XOAUTH2 " + b64("user=user\x01\x01")).BAD("A007")
		c.C("A008 AUTHENTICATE XOAUTH2 " + xoauth2("user", testBearerToken) + " extra").BAD("A008")

		c.C("A009 AUTHENTICATE OAUTHBEARER")
		c.S("+")
		c.C("*").BAD("A009")

		c.C("A010 AUTHENTICATE XOAUTH2 " + xoauth2("user", testBearerToken)).OK("A010")
	})
}

func TestOAuthLoginIsOffByDefault(t *testing.T) {
	runOneToOneTest(t, defaultServerOptions(t), func(c *testConnection, s *testSession) {
		s.conns[s.userIDs["user"]].SetBearerToken(testBearerToken)

		c.C("A001 CAPABILITY")
		c.S(`* CAPABILITY AUTH=PLAIN ID IDLE IMAP4rev1 LITERAL+ SASL-IR STARTTLS`)
		c.OK("A001")

		c.C("A002 AUTHENTICATE OAUTHBEARER " + oauthBearer("user", testBearerToken)).BAD("A002")
		c.C("A003 AUTHENTICATE XOAUTH2 " + xoauth2("user", testBearerToken)).BAD("A003")
		c.C("A004 LOGIN user pass").OK("A004")
	})
}

func TestOAuthLoginIsOffWhenAuthenticateIsDisabled(t *testing.T) {
	runOneToOneTest(t, defaultServerOptions(t, withOAuthLogin(), withDisableIMAPAuthenticate()), func(c *testConnection, s *testSession) {
		s.conns[s.userIDs["user"]].SetBearerToken(testBearerToken)

		c.C("A001 CAPABILITY")
		c.S(`* CAPABILITY ID IDLE IMAP4rev1 LITERAL+ STARTTLS`)
		c.OK("A001")

		c.C("A002 AUTHENTICATE XOAUTH2 " + xoauth2("user", testBearerToken)).BAD("A002")
	})
}

func TestOAuthLoginNeedsATokenOnTheConnector(t *testing.T) {
	// The dummy connector accepts no token until one is set.
	runOneToOneTest(t, defaultServerOptions(t, withOAuthLogin()), func(c *testConnection, _ *testSession) {
		c.C("A001 AUTHENTICATE OAUTHBEARER " + oauthBearer("user", testBearerToken)).NO("A001")
		c.C("A002 AUTHENTICATE OAUTHBEARER " + oauthBearer("user", "")).BAD("A002")
		c.C("A003 AUTHENTICATE PLAIN " + base64AuthString("user", "pass")).OK("A003")
	})
}

func TestParserAuthenticateBearer(t *testing.T) {
	bearer := command.WithBearerAuthentication()

	cmd, err := newCommandParser([]string{"tag AUTHENTICATE OAUTHBEARER " + oauthBearer("user=2Cname=3Dx@example.com", "token")}, bearer).Parse()
	require.NoError(t, err)
	require.Equal(t, command.Command{Tag: "tag", Payload: &command.AuthenticateBearer{
		Mechanism: "OAUTHBEARER",
		UserID:    "user,name=x@example.com",
		Token:     "token",
	}}, cmd)

	cmd, err = newCommandParser([]string{"tag AUTHENTICATE xoauth2 " + xoauth2("user@example.com", "token")}, bearer).Parse()
	require.NoError(t, err)
	require.Equal(t, command.Command{Tag: "tag", Payload: &command.AuthenticateBearer{
		Mechanism: "XOAUTH2",
		UserID:    "user@example.com",
		Token:     "token",
	}}, cmd)

	// Neither form of the command shows the token.
	require.NotContains(t, cmd.Payload.String(), "token")
	require.Equal(t, "AUTHENTICATE <AUTH_DATA>", cmd.Payload.SanitizedString())

	continuations := 0

	cmd, err = newCommandParser([]string{"tag AUTHENTICATE OAUTHBEARER", oauthBearer("", "token")}, bearer, countContinuations(&continuations)).Parse()
	require.NoError(t, err)
	require.Equal(t, 1, continuations)
	require.Equal(t, &command.AuthenticateBearer{Mechanism: "OAUTHBEARER", Token: "token"}, cmd.Payload)

	// Without the option the mechanisms are unknown.
	_, err = parseCommand("tag AUTHENTICATE XOAUTH2 " + xoauth2("user", "token"))
	require.Error(t, err)
}

package testserver

import (
	"net/http"
	"net/url"
	"sync"
)

// FakeOidc holds OAuth state for acceptance tests.
type FakeOidc struct {
	mu  sync.Mutex
	url string
}

func (s *FakeOidc) LockUnlock() func() {
	if s == nil {
		panic("LockUnlock called on nil FakeOidc")
	}
	s.mu.Lock()
	return func() { s.mu.Unlock() }
}

func (s *FakeOidc) OidcEndpoints() Response {
	return Response{
		Body: map[string]string{
			"authorization_endpoint": s.url + "/oidc/v1/authorize",
			"token_endpoint":         s.url + "/oidc/v1/token",
		},
	}
}

func (s *FakeOidc) OidcAuthorize(req Request) Response {
	defer s.LockUnlock()()

	redirectURI, err := url.Parse(req.URL.Query().Get("redirect_uri"))
	if err != nil {
		return Response{
			StatusCode: http.StatusBadRequest,
			Body:       err.Error(),
		}
	}

	// Compile query parameters for the redirect URL.
	q := make(url.Values)

	// Include an opaque authorization code that will be used to exchange for a token.
	q.Set("code", "oauth-code")

	// Include the state parameter from the original request.
	q.Set("state", req.URL.Query().Get("state"))

	// Update the redirect URI with the new query parameters.
	redirectURI.RawQuery = q.Encode()

	return Response{
		StatusCode: http.StatusMovedPermanently,
		Headers: map[string][]string{
			"Location": {redirectURI.String()},
		},
	}
}

func (s *FakeOidc) OidcToken(req Request) Response {
	return Response{
		Body: map[string]string{
			"access_token": "oauth-token",
			"expires_in":   "3600",
			"scope":        "all-apis",
			"token_type":   "Bearer",
		},
	}
}

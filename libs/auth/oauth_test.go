package auth

import (
	"context"
	"crypto/tls"
	_ "embed"
	"fmt"
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/databricks/databricks-sdk-go/client"
	"github.com/databricks/databricks-sdk-go/httpclient"
	"github.com/databricks/databricks-sdk-go/httpclient/fixtures"
	"github.com/databricks/databricks-sdk-go/qa"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2"
)

func TestOidcEndpointsForAccounts(t *testing.T) {
	p := &PersistentAuth{
		Host:      "abc",
		AccountID: "xyz",
	}
	defer p.Close()
	s, err := p.oidcEndpoints(context.Background())
	assert.NoError(t, err)
	assert.Equal(t, "https://abc/oidc/accounts/xyz/v1/authorize", s.AuthorizationEndpoint)
	assert.Equal(t, "https://abc/oidc/accounts/xyz/v1/token", s.TokenEndpoint)
}

func TestOidcForWorkspace(t *testing.T) {
	p := &PersistentAuth{
		Host: "abc",
		http: httpclient.NewApiClient(httpclient.ClientConfig{
			Transport: fixtures.MappingTransport{
				"GET /oidc/.well-known/oauth-authorization-server": {
					Status: 200,
					Response: map[string]string{
						"authorization_endpoint": "a",
						"token_endpoint":         "b",
					},
				},
			},
		}),
	}
	defer p.Close()
	endpoints, err := p.oidcEndpoints(context.Background())
	assert.NoError(t, err)
	assert.Equal(t, "a", endpoints.AuthorizationEndpoint)
	assert.Equal(t, "b", endpoints.TokenEndpoint)
}

type tokenCacheMock struct {
	store  func(key string, t *oauth2.Token) error
	lookup func(key string) (*oauth2.Token, error)
}

func (m *tokenCacheMock) Store(key string, t *oauth2.Token) error {
	if m.store == nil {
		panic("no store mock")
	}
	return m.store(key, t)
}

func (m *tokenCacheMock) Lookup(key string) (*oauth2.Token, error) {
	if m.lookup == nil {
		panic("no lookup mock")
	}
	return m.lookup(key)
}

func TestLoad(t *testing.T) {
	p := &PersistentAuth{
		Host:      "abc",
		AccountID: "xyz",
		cache: &tokenCacheMock{
			lookup: func(key string) (*oauth2.Token, error) {
				assert.Equal(t, "https://abc/oidc/accounts/xyz", key)
				return &oauth2.Token{
					AccessToken: "bcd",
					Expiry:      time.Now().Add(1 * time.Minute),
				}, nil
			},
		},
	}
	defer p.Close()
	tok, err := p.Load(context.Background())
	assert.NoError(t, err)
	assert.Equal(t, "bcd", tok.AccessToken)
	assert.Equal(t, "", tok.RefreshToken)
}

func useInsecureOAuthHttpClientForTests(ctx context.Context) context.Context {
	return context.WithValue(ctx, oauth2.HTTPClient, &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		},
	})
}

func TestLoadRefresh(t *testing.T) {
	qa.HTTPFixtures{
		{
			Method:   "POST",
			Resource: "/oidc/accounts/xyz/v1/token",
			Response: `access_token=refreshed&refresh_token=def`,
		},
	}.ApplyClient(t, func(ctx context.Context, c *client.DatabricksClient) {
		ctx = useInsecureOAuthHttpClientForTests(ctx)
		expectedKey := c.Config.Host + "/oidc/accounts/xyz"
		p := &PersistentAuth{
			Host:      c.Config.Host,
			AccountID: "xyz",
			cache: &tokenCacheMock{
				lookup: func(key string) (*oauth2.Token, error) {
					assert.Equal(t, expectedKey, key)
					return &oauth2.Token{
						AccessToken:  "expired",
						RefreshToken: "cde",
						Expiry:       time.Now().Add(-1 * time.Minute),
					}, nil
				},
				store: func(key string, tok *oauth2.Token) error {
					assert.Equal(t, expectedKey, key)
					assert.Equal(t, "def", tok.RefreshToken)
					return nil
				},
			},
		}
		defer p.Close()
		tok, err := p.Load(ctx)
		assert.NoError(t, err)
		assert.Equal(t, "refreshed", tok.AccessToken)
		assert.Equal(t, "", tok.RefreshToken)
	})
}

func TestChallenge(t *testing.T) {
	qa.HTTPFixtures{
		{
			Method:   "POST",
			Resource: "/oidc/accounts/xyz/v1/token",
			Response: `access_token=__THAT__&refresh_token=__SOMETHING__`,
		},
	}.ApplyClient(t, func(ctx context.Context, c *client.DatabricksClient) {
		ctx = useInsecureOAuthHttpClientForTests(ctx)
		expectedKey := c.Config.Host + "/oidc/accounts/xyz"

		browserOpened := make(chan string)
		p := &PersistentAuth{
			Host:      c.Config.Host,
			AccountID: "xyz",
			browser: func(redirect string) error {
				u, err := url.ParseRequestURI(redirect)
				if err != nil {
					return err
				}
				assert.Equal(t, "/oidc/accounts/xyz/v1/authorize", u.Path)
				// for now we're ignoring asserting the fields of the redirect
				query := u.Query()
				browserOpened <- query.Get("state")
				return nil
			},
			cache: &tokenCacheMock{
				store: func(key string, tok *oauth2.Token) error {
					assert.Equal(t, expectedKey, key)
					assert.Equal(t, "__SOMETHING__", tok.RefreshToken)
					return nil
				},
			},
		}
		defer p.Close()

		errc := make(chan error)
		go func() {
			errc <- p.Challenge(ctx)
		}()

		state := <-browserOpened
		resp, err := http.Get(fmt.Sprintf("http://%s?code=__THIS__&state=%s", appRedirectAddr, state))
		require.NoError(t, err)
		defer resp.Body.Close()
		assert.Equal(t, 200, resp.StatusCode)

		err = <-errc
		assert.NoError(t, err)
	})
}

func TestChallengeFailed(t *testing.T) {
	qa.HTTPFixtures{}.ApplyClient(t, func(ctx context.Context, c *client.DatabricksClient) {
		ctx = useInsecureOAuthHttpClientForTests(ctx)

		browserOpened := make(chan string)
		p := &PersistentAuth{
			Host:      c.Config.Host,
			AccountID: "xyz",
			browser: func(redirect string) error {
				u, err := url.ParseRequestURI(redirect)
				if err != nil {
					return err
				}
				assert.Equal(t, "/oidc/accounts/xyz/v1/authorize", u.Path)
				// for now we're ignoring asserting the fields of the redirect
				query := u.Query()
				browserOpened <- query.Get("state")
				return nil
			},
		}
		defer p.Close()

		errc := make(chan error)
		go func() {
			errc <- p.Challenge(ctx)
		}()

		<-browserOpened
		resp, err := http.Get(fmt.Sprintf(
			"http://%s?error=access_denied&error_description=Policy%%20evaluation%%20failed%%20for%%20this%%20request",
			appRedirectAddr))
		require.NoError(t, err)
		defer resp.Body.Close()
		assert.Equal(t, 400, resp.StatusCode)

		err = <-errc
		assert.EqualError(t, err, "authorize: access_denied: Policy evaluation failed for this request")
	})
}

func TestPersistentAuthCleanHost(t *testing.T) {
	for _, tcases := range []struct {
		in  string
		out string
	}{
		{"https://example.com", "https://example.com"},
		{"https://example.com/", "https://example.com"},
		{"https://example.com/path", "https://example.com"},
		{"https://example.com/path/subpath", "https://example.com"},
		{"https://example.com/path?query=1", "https://example.com"},
		{"https://example.com/path?query=1&other=2", "https://example.com"},
		{"https://example.com/path#fragment", "https://example.com"},
		{"https://example.com/path?query=1#fragment", "https://example.com"},
		{"https://example.com/path?query=1&other=2#fragment", "https://example.com"},
		{"https://example.com/path/subpath?query=1", "https://example.com"},
		{"https://example.com/path/subpath?query=1&other=2", "https://example.com"},
		{"https://example.com/path/subpath#fragment", "https://example.com"},
		{"https://example.com/path/subpath?query=1#fragment", "https://example.com"},
		{"https://example.com/path/subpath?query=1&other=2#fragment", "https://example.com"},
		{"https://example.com/path?query=1%20value&other=2%20value", "https://example.com"},
		{"http://example.com/path/subpath?query=1%20value&other=2%20value", "http://example.com"},

		// URLs without scheme should be left as is
		{"abc", "abc"},
		{"abc.com/def", "abc.com/def"},
	} {
		p := &PersistentAuth{
			Host: tcases.in,
		}
		p.cleanHost()
		assert.Equal(t, tcases.out, p.Host)
	}
}

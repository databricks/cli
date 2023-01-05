package auth

import (
	"context"
	"crypto/tls"
	_ "embed"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/databricks/databricks-sdk-go/client"
	"github.com/databricks/databricks-sdk-go/qa"
	"github.com/stretchr/testify/assert"
	"golang.org/x/oauth2"
)

func TestNewPersistentOAuth(t *testing.T) {
	type thing struct {
		host string
		want *PersistentAuth
		err  string
	}
	tests := []thing{
		{
			err: "host cannot be empty",
		},
		{
			host: "localhost",
			want: &PersistentAuth{
				Host: "localhost",
			},
		},
		{
			host: "https://localhost",
			want: &PersistentAuth{
				Host: "localhost",
			},
		},
		{
			host: "https://accounts.any",
			err:  "path does not end in UUID: ",
		},
		{
			host: "https://accounts.any/a5115405-77bb-4fc3-8cfa-6963ca3dde04",
			want: &PersistentAuth{
				Host:      "accounts.any",
				AccountID: "a5115405-77bb-4fc3-8cfa-6963ca3dde04",
			},
		},
		{
			host: "a5115405-77bb-4fc3-8cfa-6963ca3dde04",
			want: &PersistentAuth{
				Host:      "accounts.cloud.databricks.com",
				AccountID: "a5115405-77bb-4fc3-8cfa-6963ca3dde04",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.host, func(t *testing.T) {
			got, err := NewPersistentOAuth(tt.host)
			if tt.err != "" {
				assert.EqualError(t, err, tt.err)
			} else if err != nil {
				assert.NoError(t, err)
			} else {
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func TestOidcEndpointsForAccounts(t *testing.T) {
	p := &PersistentAuth{
		Host:      "abc",
		AccountID: "xyz",
	}
	defer p.Close()
	s, err := p.oidcEndpoints()
	assert.NoError(t, err)
	assert.Equal(t, "https://abc/oidc/accounts/xyz/v1/authorize", s.AuthorizationEndpoint)
	assert.Equal(t, "https://abc/oidc/accounts/xyz/v1/token", s.TokenEndpoint)
}

type mockGet func(url string) (*http.Response, error)

func (m mockGet) Get(url string) (*http.Response, error) {
	return m(url)
}

func TestOidcForWorkspace(t *testing.T) {
	p := &PersistentAuth{
		Host: "abc",
		http: mockGet(func(url string) (*http.Response, error) {
			assert.Equal(t, "https://abc/oidc/.well-known/oauth-authorization-server", url)
			return &http.Response{
				StatusCode: 200,
				Body: io.NopCloser(strings.NewReader(`{
					"authorization_endpoint": "a",
					"token_endpoint": "b"
				}`)),
			}, nil
		}),
	}
	defer p.Close()
	endpoints, err := p.oidcEndpoints()
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
		expectedKey := fmt.Sprintf("%s/oidc/accounts/xyz", c.Config.Host)
		p := &PersistentAuth{
			Host:      strings.TrimPrefix(c.Config.Host, "http://"),
			AccountID: "xyz",
			scheme:    "http",
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
		expectedKey := fmt.Sprintf("%s/oidc/accounts/xyz", c.Config.Host)

		browserOpened := make(chan string)
		p := &PersistentAuth{
			Host:      strings.TrimPrefix(c.Config.Host, "http://"),
			AccountID: "xyz",
			scheme:    "http",
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
		assert.NoError(t, err)
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
			Host:      strings.TrimPrefix(c.Config.Host, "http://"),
			AccountID: "xyz",
			scheme:    "http",
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
		assert.NoError(t, err)
		assert.Equal(t, 400, resp.StatusCode)

		err = <-errc
		assert.EqualError(t, err, "authorize: access_denied: Policy evaluation failed for this request")
	})
}

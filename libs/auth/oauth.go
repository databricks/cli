package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	_ "embed"
	"encoding/base64"
	"errors"
	"fmt"
	"net"
	"net/url"
	"strings"
	"time"

	"github.com/databricks/cli/libs/auth/cache"
	"github.com/databricks/databricks-sdk-go/httpclient"
	"github.com/databricks/databricks-sdk-go/retries"
	"github.com/pkg/browser"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/authhandler"
)

var apiClientForOauth int

func WithApiClientForOAuth(ctx context.Context, c *httpclient.ApiClient) context.Context {
	return context.WithValue(ctx, &apiClientForOauth, c)
}

func GetApiClientForOAuth(ctx context.Context) *httpclient.ApiClient {
	c, ok := ctx.Value(&apiClientForOauth).(*httpclient.ApiClient)
	if !ok {
		return httpclient.NewApiClient(httpclient.ClientConfig{})
	}
	return c
}

const (
	// these values are predefined by Databricks as a public client
	// and is specific to this application only. Using these values
	// for other applications is not allowed.
	appClientID     = "databricks-cli"
	appRedirectAddr = "localhost:8020"

	// maximum amount of time to acquire listener on appRedirectAddr
	listenerTimeout = 45 * time.Second
)

var ( // Databricks SDK API: `databricks OAuth is not` will be checked for presence
	ErrOAuthNotSupported = errors.New("databricks OAuth is not supported for this host")
	ErrNotConfigured     = errors.New("databricks OAuth is not configured for this host")
	ErrFetchCredentials  = errors.New("cannot fetch credentials")
)

type PersistentAuth struct {
	Host      string
	AccountID string

	http    *httpclient.ApiClient
	cache   cache.TokenCache
	ln      net.Listener
	browser func(string) error
}

func (a *PersistentAuth) SetApiClient(h *httpclient.ApiClient) {
	a.http = h
}

func (a *PersistentAuth) Load(ctx context.Context) (*oauth2.Token, error) {
	err := a.init(ctx)
	if err != nil {
		return nil, fmt.Errorf("init: %w", err)
	}
	// lookup token identified by host (and possibly the account id)
	key := a.key()
	t, err := a.cache.Lookup(key)
	if err != nil {
		return nil, fmt.Errorf("cache: %w", err)
	}
	// early return for valid tokens
	if t.Valid() {
		// do not print refresh token to end-user
		t.RefreshToken = ""
		return t, nil
	}
	// OAuth2 config is invoked only for expired tokens to speed up
	// the happy path in the token retrieval
	cfg, err := a.oauth2Config(ctx)
	if err != nil {
		return nil, err
	}
	// make OAuth2 library use our client
	ctx = a.http.InContextForOAuth2(ctx)
	// eagerly refresh token
	refreshed, err := cfg.TokenSource(ctx, t).Token()
	if err != nil {
		return nil, fmt.Errorf("token refresh: %w", err)
	}
	err = a.cache.Store(key, refreshed)
	if err != nil {
		return nil, fmt.Errorf("cache refresh: %w", err)
	}
	// do not print refresh token to end-user
	refreshed.RefreshToken = ""
	return refreshed, nil
}

func (a *PersistentAuth) ProfileName() string {
	if a.AccountID != "" {
		return "ACCOUNT-" + a.AccountID
	}
	host := strings.TrimPrefix(a.Host, "https://")
	split := strings.Split(host, ".")
	return split[0]
}

func (a *PersistentAuth) Challenge(ctx context.Context) error {
	err := a.init(ctx)
	if err != nil {
		return fmt.Errorf("init: %w", err)
	}
	cfg, err := a.oauth2Config(ctx)
	if err != nil {
		return err
	}
	cb, err := newCallback(ctx, a)
	if err != nil {
		return fmt.Errorf("callback server: %w", err)
	}
	defer cb.Close()
	state, pkce := a.stateAndPKCE()
	// make OAuth2 library use our client
	ctx = a.http.InContextForOAuth2(ctx)
	ts := authhandler.TokenSourceWithPKCE(ctx, cfg, state, cb.Handler, pkce)
	t, err := ts.Token()
	if err != nil {
		return fmt.Errorf("authorize: %w", err)
	}
	// cache token identified by host (and possibly the account id)
	err = a.cache.Store(a.key(), t)
	if err != nil {
		return fmt.Errorf("store: %w", err)
	}
	return nil
}

// This function cleans up the host URL by only retaining the scheme and the host.
// This function thus removes any path, query arguments, or fragments from the URL.
func (a *PersistentAuth) cleanHost() {
	parsedHost, err := url.Parse(a.Host)
	if err != nil {
		return
	}
	// when either host or scheme is empty, we don't want to clean it. This is because
	// the Go url library parses a raw "abc" string as the path of a URL and cleaning
	// it will return thus return an empty string.
	if parsedHost.Host == "" || parsedHost.Scheme == "" {
		return
	}
	host := url.URL{
		Scheme: parsedHost.Scheme,
		Host:   parsedHost.Host,
	}
	a.Host = host.String()
}

func (a *PersistentAuth) init(ctx context.Context) error {
	if a.Host == "" && a.AccountID == "" {
		return ErrFetchCredentials
	}
	if a.http == nil {
		a.http = GetApiClientForOAuth(ctx)
	}
	if a.cache == nil {
		a.cache = cache.GetTokenCache(ctx)
	}
	if a.browser == nil {
		a.browser = browser.OpenURL
	}

	a.cleanHost()

	// try acquire listener, which we also use as a machine-local
	// exclusive lock to prevent token cache corruption in the scope
	// of developer machine, where this command runs.
	listener, err := retries.Poll(ctx, listenerTimeout,
		func() (*net.Listener, *retries.Err) {
			var lc net.ListenConfig
			l, err := lc.Listen(ctx, "tcp", appRedirectAddr)
			if err != nil {
				return nil, retries.Continue(err)
			}
			return &l, nil
		})
	if err != nil {
		return fmt.Errorf("listener: %w", err)
	}
	a.ln = *listener
	return nil
}

func (a *PersistentAuth) Close() error {
	if a.ln == nil {
		return nil
	}
	return a.ln.Close()
}

func (a *PersistentAuth) oidcEndpoints(ctx context.Context) (*oauthAuthorizationServer, error) {
	prefix := a.key()
	if a.AccountID != "" {
		return &oauthAuthorizationServer{
			AuthorizationEndpoint: prefix + "/v1/authorize",
			TokenEndpoint:         prefix + "/v1/token",
		}, nil
	}
	var oauthEndpoints oauthAuthorizationServer
	oidc := prefix + "/oidc/.well-known/oauth-authorization-server"
	err := a.http.Do(ctx, "GET", oidc, httpclient.WithResponseUnmarshal(&oauthEndpoints))
	if err != nil {
		return nil, fmt.Errorf("fetch .well-known: %w", err)
	}
	var httpErr *httpclient.HttpError
	if errors.As(err, &httpErr) && httpErr.StatusCode != 200 {
		return nil, ErrOAuthNotSupported
	}
	return &oauthEndpoints, nil
}

func (a *PersistentAuth) oauth2Config(ctx context.Context) (*oauth2.Config, error) {
	// in this iteration of CLI, we're using all scopes by default,
	// because tools like CLI and Terraform do use all apis. This
	// decision may be reconsidered later, once we have a proper
	// taxonomy of all scopes ready and implemented.
	scopes := []string{
		"offline_access",
		"all-apis",
	}
	endpoints, err := a.oidcEndpoints(ctx)
	if err != nil {
		return nil, fmt.Errorf("oidc: %w", err)
	}
	return &oauth2.Config{
		ClientID: appClientID,
		Endpoint: oauth2.Endpoint{
			AuthURL:   endpoints.AuthorizationEndpoint,
			TokenURL:  endpoints.TokenEndpoint,
			AuthStyle: oauth2.AuthStyleInParams,
		},
		RedirectURL: "http://" + appRedirectAddr,
		Scopes:      scopes,
	}, nil
}

// key is currently used for two purposes: OIDC URL prefix and token cache key.
// once we decide to start storing scopes in the token cache, we should change
// this approach.
func (a *PersistentAuth) key() string {
	a.Host = strings.TrimSuffix(a.Host, "/")
	if !strings.HasPrefix(a.Host, "http") {
		a.Host = "https://" + a.Host
	}
	if a.AccountID != "" {
		return fmt.Sprintf("%s/oidc/accounts/%s", a.Host, a.AccountID)
	}
	return a.Host
}

func (a *PersistentAuth) stateAndPKCE() (string, *authhandler.PKCEParams) {
	verifier := a.randomString(64)
	verifierSha256 := sha256.Sum256([]byte(verifier))
	challenge := base64.RawURLEncoding.EncodeToString(verifierSha256[:])
	return a.randomString(16), &authhandler.PKCEParams{
		Challenge:       challenge,
		ChallengeMethod: "S256",
		Verifier:        verifier,
	}
}

func (a *PersistentAuth) randomString(size int) string {
	raw := make([]byte, size)
	_, _ = rand.Read(raw)
	return base64.RawURLEncoding.EncodeToString(raw)
}

type oauthAuthorizationServer struct {
	AuthorizationEndpoint string `json:"authorization_endpoint"` // ../v1/authorize
	TokenEndpoint         string `json:"token_endpoint"`         // ../v1/token
}

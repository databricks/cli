package auth

import (
	"context"
	"crypto/sha256"
	_ "embed"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/authhandler"
)

var ( // Databricks SDK API: `databricks OAuth is not` will be checked for presence
	ErrOAuthNotSupported = errors.New("databricks OAuth is not supported for this host")
	ErrNotConfigured     = errors.New("databricks OAuth is not configured for this host")
)

var uuidRegex = regexp.MustCompile(`(?mi)^[a-f0-9]{8}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{12}$`)

type PersistentAuth struct {
	Host      string
	AccountID string
	mu        portLocker
}

func NewPersistentOAuth(host string) (*PersistentAuth, error) {
	if host == "" {
		return nil, fmt.Errorf("host cannot be empty")
	}
	if uuidRegex.Match([]byte(host)) {
		// Example: bricks login a5115405-77bb-4fc3-8cfa-6963ca3dde04
		return &PersistentAuth{
			Host:      "accounts.cloud.databricks.com",
			AccountID: host,
		}, nil
	}
	parsedUrl, err := url.Parse(host)
	if err != nil {
		return nil, err
	}
	if parsedUrl.Host == "" {
		// Example: bricks login XYZ.cloud.databricks.com
		return &PersistentAuth{
			Host: host,
		}, nil
	}
	if strings.HasPrefix(parsedUrl.Host, "accounts.") {
		shouldBeUuid := filepath.Base(parsedUrl.Path)
		if !uuidRegex.Match([]byte(shouldBeUuid)) {
			return nil, fmt.Errorf("path does not end in UUID: %s", parsedUrl.Path)
		}
		// Example: bricks login https://accounts.../oidc/accounts/a5115405-77bb-4fc3-8cfa-6963ca3dde04
		return &PersistentAuth{
			Host:      parsedUrl.Host,
			AccountID: shouldBeUuid,
		}, nil
	}
	// Example: bricks login https://XYZ.cloud.databricks.com
	return &PersistentAuth{
		Host: parsedUrl.Host,
	}, nil
}

func (a *PersistentAuth) oidcEndpoints() (*oauthAuthorizationServer, error) {
	if a.AccountID != "" {
		prefix := fmt.Sprintf("https://%s/oidc/accounts/%s", a.Host, a.AccountID)
		return &oauthAuthorizationServer{
			AuthorizationEndpoint: fmt.Sprintf("%s/v1/authorize", prefix),
			TokenEndpoint:         fmt.Sprintf("%s/v1/token", prefix),
		}, nil
	}
	oidc := fmt.Sprintf("https://%s/oidc/.well-known/oauth-authorization-server", a.Host)
	oidcResponse, err := http.Get(oidc)
	if err != nil {
		return nil, fmt.Errorf("fetch .well-known: %w", err)
	}
	if oidcResponse.StatusCode != 200 {
		return nil, ErrOAuthNotSupported
	}
	if oidcResponse.Body == nil {
		return nil, fmt.Errorf("fetch .well-known: empty body")
	}
	defer oidcResponse.Body.Close()
	raw, err := io.ReadAll(oidcResponse.Body)
	if err != nil {
		return nil, fmt.Errorf("read .well-known: %w", err)
	}
	var oauthEndpoints oauthAuthorizationServer
	err = json.Unmarshal(raw, &oauthEndpoints)
	if err != nil {
		return nil, fmt.Errorf("parse .well-known: %w", err)
	}
	return &oauthEndpoints, nil
}

func (a *PersistentAuth) oauth2Config() (*oauth2.Config, error) {
	scopes := []string{ // default
		"offline_access",
		"unity-catalog",
		"accounts",
		"clusters",
		"mlflow",
		"scim",
		"sql",
	}
	if a.AccountID != "" {
		// doesn't seem to work yet...
		scopes = []string{
			"accounts",
		}
	}
	endpoints, err := a.oidcEndpoints()
	if err != nil {
		return nil, fmt.Errorf("oidc: %w", err)
	}
	return &oauth2.Config{
		ClientID: "databricks-cli",
		Endpoint: oauth2.Endpoint{
			AuthURL:   endpoints.AuthorizationEndpoint,
			TokenURL:  endpoints.TokenEndpoint,
			AuthStyle: oauth2.AuthStyleInParams,
		},
		RedirectURL: fmt.Sprintf("http://%s", appRedirectAddr),
		Scopes:      scopes,
	}, nil
}

func (a *PersistentAuth) location() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("home: %w", err)
	}
	// we can also store all cached credentials in one single file,
	// like ~/.azure/msal_token_cache.json for az login - in our case it'll be
	// something like ~/.databricks/token-cache.json
	return fmt.Sprintf("%s/.databricks/auth/%s.json", home, a.Host), nil
}

func (a *PersistentAuth) store(ctx context.Context, t *oauth2.Token) error {
	raw, err := json.MarshalIndent(t, "", "  ")
	if err != nil {
		return fmt.Errorf("token to json: %w", err)
	}
	loc, err := a.location()
	if err != nil {
		return fmt.Errorf("token cache: %w", err)
	}
	tokenCacheDir := filepath.Dir(loc)
	_, err = os.Stat(tokenCacheDir)
	if os.IsNotExist(err) {
		err = os.MkdirAll(tokenCacheDir, 0o600)
		if err != nil {
			return fmt.Errorf("token cache mkdir: %w", err)
		}
	}
	return os.WriteFile(loc, raw, 0o600)
}

func (a *PersistentAuth) Load(ctx context.Context) (*oauth2.Token, error) {
	err := a.mu.Lock(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to acquire lock: %w", err)
	}
	defer a.mu.Unlock()
	loc, err := a.location()
	if err != nil {
		return nil, fmt.Errorf("token cache: %w", err)
	}
	raw, err := os.ReadFile(loc)
	if os.IsNotExist(err) {
		return nil, ErrNotConfigured
	}
	if err != nil {
		return nil, fmt.Errorf("token cache read: %w", err)
	}
	var t oauth2.Token
	err = json.Unmarshal(raw, &t)
	if err != nil {
		return nil, fmt.Errorf("corrput token cache: %w", err)
	}
	if t.Valid() {
		return &t, nil
	}
	config, err := a.oauth2Config()
	if err != nil {
		return nil, err
	}
	refreshed, err := config.TokenSource(ctx, &t).Token()
	if err != nil {
		return nil, fmt.Errorf("token refresh: %w", err)
	}
	err = a.store(ctx, refreshed)
	if err != nil {
		return nil, fmt.Errorf("store in token cache: %w", err)
	}
	// do not print refresh token to end-user
	refreshed.RefreshToken = ""
	return refreshed, nil
}

func (a *PersistentAuth) Challenge(ctx context.Context) error {
	config, err := a.oauth2Config()
	if err != nil {
		return err
	}
	cb, err := newCallback(ctx, a)
	if err != nil {
		return fmt.Errorf("callback server: %w", err)
	}
	defer cb.Close()
	state, pkce := a.stateAndPKCE()
	ts := authhandler.TokenSourceWithPKCE(ctx, config, state,
		cb.AuthorizationHandler, pkce)
	t, err := ts.Token()
	if err != nil {
		return fmt.Errorf("authorize: %w", err)
	}
	err = a.mu.Lock(ctx)
	if err != nil {
		return fmt.Errorf("failed to acquire lock: %w", err)
	}
	defer a.mu.Unlock()
	return a.store(ctx, t)
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
	rand.Seed(time.Now().UnixNano())
	raw := make([]byte, size)
	_, _ = rand.Read(raw)
	return base64.RawURLEncoding.EncodeToString(raw)
}

type oauthAuthorizationServer struct {
	AuthorizationEndpoint string `json:"authorization_endpoint"` // ../v1/authorize
	TokenEndpoint         string `json:"token_endpoint"`         // ../v1/token
}

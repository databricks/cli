package u2m

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"golang.org/x/oauth2"
)

func TestDeriveHostFromIssuer(t *testing.T) {
	tests := []struct {
		name    string
		issuer  string
		want    string
		wantErr bool
	}{
		{
			name:   "workspace oidc",
			issuer: "https://adb-xxx.azuredatabricks.test/oidc",
			want:   "https://adb-xxx.azuredatabricks.test",
		},
		{
			name:   "workspace cloud",
			issuer: "https://workspace.cloud.databricks.test/oidc",
			want:   "https://workspace.cloud.databricks.test",
		},
		{
			name:   "spog with account path",
			issuer: "https://nike.databricks.test/oidc/accounts/xxx",
			want:   "https://nike.databricks.test",
		},
		{
			name:    "empty issuer",
			issuer:  "",
			wantErr: true,
		},
		{
			name:    "http scheme",
			issuer:  "http://insecure.net/oidc",
			wantErr: true,
		},
		{
			name:    "no host",
			issuer:  "https:///oidc",
			wantErr: true,
		},
		{
			name:   "http localhost allowed",
			issuer: "http://127.0.0.1:12345/oidc",
			want:   "http://127.0.0.1:12345",
		},
		{
			name:    "http non-local host rejected",
			issuer:  "http://127.0.0.1.attacker.tld/oidc",
			wantErr: true,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := DeriveHostFromIssuer(tc.issuer)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("DeriveHostFromIssuer(%q): want error, got nil", tc.issuer)
				}
				return
			}
			if err != nil {
				t.Fatalf("DeriveHostFromIssuer(%q): unexpected error: %v", tc.issuer, err)
			}
			if got != tc.want {
				t.Errorf("DeriveHostFromIssuer(%q) = %q, want %q", tc.issuer, got, tc.want)
			}
		})
	}
}

func TestDeriveTokenEndpoint(t *testing.T) {
	tests := []struct {
		name   string
		issuer string
		want   string
	}{
		{
			name:   "standard",
			issuer: "https://adb-xxx.net/oidc",
			want:   "https://adb-xxx.net/oidc/v1/token",
		},
		{
			name:   "trailing slash",
			issuer: "https://adb-xxx.net/oidc/",
			want:   "https://adb-xxx.net/oidc/v1/token",
		},
		{
			name:   "account path",
			issuer: "https://nike.databricks.test/oidc/accounts/abc123",
			want:   "https://nike.databricks.test/oidc/accounts/abc123/v1/token",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := DeriveTokenEndpoint(tc.issuer)
			if got != tc.want {
				t.Errorf("DeriveTokenEndpoint(%q) = %q, want %q", tc.issuer, got, tc.want)
			}
		})
	}
}

func TestBuildDiscoveryAuthorizeURL(t *testing.T) {
	pkce := PKCEParams{
		Challenge:       "test-challenge",
		ChallengeMethod: "S256",
		Verifier:        "test-verifier",
	}
	scopes := []string{"offline_access", "all-apis"}
	redirectAddr := "localhost:8020"
	state := "test-state"

	got := BuildDiscoveryAuthorizeURL(redirectAddr, state, pkce, scopes)

	// Parse the top-level URL.
	u, err := url.Parse(got)
	if err != nil {
		t.Fatalf("parsing URL: %v", err)
	}
	if u.Scheme != "https" || u.Host != "login.databricks.com" {
		t.Errorf("want host https://login.databricks.com, got %s://%s", u.Scheme, u.Host)
	}

	// Extract and decode destination_url.
	destURL := u.Query().Get("destination_url")
	if destURL == "" {
		t.Fatal("destination_url query param is empty")
	}

	// Parse the destination_url as a relative URL with query params.
	destParsed, err := url.Parse(destURL)
	if err != nil {
		t.Fatalf("parsing destination_url: %v", err)
	}
	if destParsed.Path != "/oidc/v1/authorize" {
		t.Errorf("destination_url path = %q, want %q", destParsed.Path, "/oidc/v1/authorize")
	}

	// Verify all expected OAuth params.
	q := destParsed.Query()
	expectations := map[string]string{
		"client_id":             appClientID,
		"redirect_uri":          "http://localhost:8020",
		"response_type":         "code",
		"scope":                 "offline_access all-apis",
		"state":                 "test-state",
		"code_challenge":        "test-challenge",
		"code_challenge_method": "S256",
	}
	for key, want := range expectations {
		if got := q.Get(key); got != want {
			t.Errorf("destination_url param %q = %q, want %q", key, got, want)
		}
	}
}

func TestBuildDiscoveryAuthorizeURL_HostOverride(t *testing.T) {
	pkce := PKCEParams{
		Challenge:       "c",
		ChallengeMethod: "S256",
		Verifier:        "v",
	}
	scopes := []string{"offline_access", "all-apis"}
	tests := []struct {
		name string
		host string
	}{
		{
			name: "custom host",
			host: "https://login.dev.databricks.test",
		},
		{
			name: "custom host with trailing slash",
			host: "https://login.dev.databricks.test/",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := buildDiscoveryAuthorizeURL(tc.host, "localhost:8020", "s", pkce, scopes, appClientID, "")
			u, err := url.Parse(got)
			if err != nil {
				t.Fatalf("parsing URL: %v", err)
			}
			if u.Host != "login.dev.databricks.test" {
				t.Errorf("host = %q, want login.dev.databricks.test", u.Host)
			}
		})
	}
}

func TestBuildDiscoveryAuthorizeURL_Target(t *testing.T) {
	pkce := PKCEParams{
		Challenge:       "c",
		ChallengeMethod: "S256",
		Verifier:        "v",
	}
	scopes := []string{"offline_access", "all-apis"}
	tests := []struct {
		name       string
		target     string
		wantTarget string
	}{
		{name: "no target", target: "", wantTarget: ""},
		{name: "account target", target: "ACCOUNT", wantTarget: "ACCOUNT"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := buildDiscoveryAuthorizeURL(defaultLoginDatabricksHost, "localhost:8020", "s", pkce, scopes, appClientID, tc.target)
			u, err := url.Parse(got)
			if err != nil {
				t.Fatalf("parsing URL: %v", err)
			}
			if g := u.Query().Get("target"); g != tc.wantTarget {
				t.Errorf("target = %q, want %q", g, tc.wantTarget)
			}
			// destination_url must still be present in every variant.
			if u.Query().Get("destination_url") == "" {
				t.Error("destination_url should be set regardless of target")
			}
		})
	}
}

func TestWithDiscoveryAccountTarget(t *testing.T) {
	var a PersistentAuth
	if a.discoveryAccountTarget {
		t.Fatal("discoveryAccountTarget should default to false")
	}
	WithDiscoveryAccountTarget()(&a)
	if !a.discoveryAccountTarget {
		t.Error("WithDiscoveryAccountTarget did not set discoveryAccountTarget")
	}
}

func TestWithDiscoveryHost_NormalizesScheme(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{name: "empty stays empty", input: "", want: ""},
		{name: "https preserved", input: "https://login.dev.databricks.test", want: "https://login.dev.databricks.test"},
		{name: "http preserved", input: "http://localhost:8080", want: "http://localhost:8080"},
		{name: "no scheme gets https", input: "login.dev.databricks.test", want: "https://login.dev.databricks.test"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var a PersistentAuth
			WithDiscoveryHost(tc.input)(&a)
			if a.discoveryHost != tc.want {
				t.Errorf("discoveryHost = %q, want %q", a.discoveryHost, tc.want)
			}
		})
	}
}

func TestDiscoveryTokenSource_Challenge(t *testing.T) {
	// Create a mock token server that responds to POST /oidc/v1/token.
	tokenServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("token server: want POST, got %s", r.Method)
		}
		if r.URL.Path != "/oidc/v1/token" {
			t.Errorf("token server: want path /oidc/v1/token, got %s", r.URL.Path)
		}
		if err := r.ParseForm(); err != nil {
			t.Fatalf("token server: parsing form: %v", err)
		}
		if got := r.Form.Get("client_id"); got != "custom-client-id" {
			t.Errorf("token server: client_id = %q, want %q", got, "custom-client-id")
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"access_token":"test-access-token","refresh_token":"test-refresh-token","token_type":"Bearer","expires_in":3600}`)
	}))
	defer tokenServer.Close()

	// The issuer will be the mock server URL + /oidc.
	issuer := tokenServer.URL + "/oidc"

	browserOpened := make(chan string, 1)
	browserMock := func(u string) error {
		browserOpened <- u
		return nil
	}

	storedTokens := map[string]*oauth2.Token{}
	cacheMock := &tokenStoreMock{
		store: func(key string, tok *oauth2.Token) error {
			storedTokens[key] = tok
			return nil
		},
	}

	arg, err := NewBasicDiscoveryOAuthArgument("test-profile")
	if err != nil {
		t.Fatalf("NewBasicDiscoveryOAuthArgument(): %v", err)
	}

	p, err := NewPersistentAuth(
		t.Context(),
		WithTokenStore(cacheMock),
		WithBrowser(browserMock),
		WithHttpClient(tokenServer.Client()),
		WithOAuthEndpointSupplier(MockOAuthEndpointSupplier{}),
		WithOAuthArgument(arg),
		WithDiscoveryLogin(),
		WithClientID("custom-client-id"),
	)
	if err != nil {
		t.Fatalf("NewPersistentAuth(): %v", err)
	}

	// Start the listener so redirectAddr is set.
	err = p.startListener(t.Context())
	if err != nil {
		t.Fatalf("startListener(): %v", err)
	}
	defer p.Close()

	dts := &discoveryTokenSource{pa: p}

	errc := make(chan error, 1)
	go func() {
		errc <- dts.challenge()
	}()

	// Wait for browser to be called and extract state from the URL.
	var state string
	select {
	case authURL := <-browserOpened:
		u, err := url.Parse(authURL)
		if err != nil {
			t.Fatalf("parsing auth URL: %v", err)
		}
		destURL := u.Query().Get("destination_url")
		dest, err := url.Parse(destURL)
		if err != nil {
			t.Fatalf("parsing destination_url: %v", err)
		}
		if got := dest.Query().Get("client_id"); got != "custom-client-id" {
			t.Errorf("authorize URL: client_id = %q, want %q", got, "custom-client-id")
		}
		state = dest.Query().Get("state")
		if state == "" {
			t.Fatal("state is empty in authorize URL")
		}
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for browser to be called")
	}

	// Fire the callback with code, state, and iss.
	callbackURL := fmt.Sprintf("http://%s?code=test-code&state=%s&iss=%s",
		p.redirectAddr, url.QueryEscape(state), url.QueryEscape(issuer))
	resp, err := http.Get(callbackURL)
	if err != nil {
		t.Fatalf("callback GET: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("callback: want status 200, got %d", resp.StatusCode)
	}

	// Wait for challenge to complete.
	select {
	case err := <-errc:
		if err != nil {
			t.Fatalf("challenge(): %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for challenge to complete")
	}

	// Verify discovered host was set.
	expectedHost, err := DeriveHostFromIssuer(issuer)
	if err != nil {
		t.Fatalf("DeriveHostFromIssuer(%q): %v", issuer, err)
	}
	if arg.GetDiscoveredHost() != expectedHost {
		t.Errorf("discovered host = %q, want %q", arg.GetDiscoveredHost(), expectedHost)
	}
	if len(storedTokens) != 1 {
		t.Fatalf("store count: want 1 key (profile), got %d", len(storedTokens))
	}
	storedToken := storedTokens["test-profile"]
	if storedToken == nil {
		t.Fatalf("stored token for profile key is nil")
	}
	if storedToken.AccessToken != "test-access-token" {
		t.Errorf("access token = %q, want %q", storedToken.AccessToken, "test-access-token")
	}
	if storedToken.RefreshToken != "test-refresh-token" {
		t.Errorf("refresh token = %q, want %q", storedToken.RefreshToken, "test-refresh-token")
	}
}

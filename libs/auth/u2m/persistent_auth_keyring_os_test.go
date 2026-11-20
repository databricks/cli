//go:build darwin || windows

package u2m

import (
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"
	"uuid"

	"github.com/databricks/cli/libs/auth/storage"
	"github.com/databricks/databricks-sdk-go/httpclient/fixtures"
	"github.com/zalando/go-keyring"
	"golang.org/x/oauth2"
)

// TestToken_RefreshesLargeTokenInOSKeyring refreshes an expired token that the
// real OS keyring holds in several items, because its access token is too
// large for one, and reads the new access token and refresh token back. It
// uses the store that the CLI uses, with its 3-second timeout per keyring
// operation.
//
// It writes to the real OS keyring and therefore runs on CI only (GitHub
// Actions sets CI=true).
func TestToken_RefreshesLargeTokenInOSKeyring(t *testing.T) {
	if os.Getenv("CI") == "" {
		t.Skip("touches the real OS keyring; runs on CI only")
	}

	// The keyring store keeps every item in this service.
	const service = "databricks-cli"
	profile := "__probe_" + uuid.New().String()
	t.Cleanup(func() {
		_ = keyring.Delete(service, profile)
		for i := range 4 {
			_ = keyring.Delete(service, profile+storage.AccessTokenPartMarker+strconv.Itoa(i))
		}
	})

	// The size of the access tokens in https://github.com/databricks/cli/issues/7707.
	const accessTokenSize = 3250
	tokenStore := storage.NewKeyringStore()
	err := tokenStore.Put(profile, storage.Entry{Token: &oauth2.Token{
		AccessToken:  strings.Repeat("a", accessTokenSize),
		RefreshToken: "first",
		Expiry:       time.Now().Add(-time.Minute),
	}})
	if err != nil {
		t.Fatalf("Put(): want no error, got %v", err)
	}

	refreshed := strings.Repeat("b", accessTokenSize)
	arg, err := NewProfileWorkspaceOAuthArgument("https://keyring-refresh.cloud.databricks.test", profile)
	if err != nil {
		t.Fatalf("NewProfileWorkspaceOAuthArgument(): want no error, got %v", err)
	}
	p, err := NewPersistentAuth(
		t.Context(),
		WithTokenStore(tokenStore),
		WithHttpClient(&http.Client{
			Transport: fixtures.SliceTransport{
				{
					Method:          "POST",
					Resource:        "/oidc/v1/token",
					ExpectedRequest: url.Values{"client_id": {"databricks-cli"}, "grant_type": {"refresh_token"}, "refresh_token": {"first"}},
					Response:        "access_token=" + refreshed + "&refresh_token=second&expires_in=3600",
					ResponseHeaders: map[string][]string{
						"Content-Type": {"application/x-www-form-urlencoded"},
					},
				},
			},
		}),
		WithOAuthEndpointSupplier(MockOAuthEndpointSupplier{}),
		WithOAuthArgument(arg),
	)
	if err != nil {
		t.Fatalf("NewPersistentAuth(): want no error, got %v", err)
	}
	defer p.Close()

	tok, err := p.Token()
	if err != nil {
		t.Fatalf("p.Token(): want no error, got %v", err)
	}
	if tok.AccessToken != refreshed {
		t.Errorf("p.Token(): want the refreshed access token, got %d bytes", len(tok.AccessToken))
	}

	if _, err := keyring.Get(service, profile+storage.AccessTokenPartMarker+"1"); err != nil {
		t.Errorf("the refreshed token was not split: %v", err)
	}
	stored, err := storage.NewKeyringStore().Lookup(profile)
	if err != nil {
		t.Fatalf("Lookup(): want no error, got %v", err)
	}
	if stored.Token.AccessToken != refreshed {
		t.Errorf("Lookup(): want the refreshed access token, got %d bytes", len(stored.Token.AccessToken))
	}
	if stored.Token.RefreshToken != "second" {
		t.Errorf("Lookup(): want refresh token 'second', got %q", stored.Token.RefreshToken)
	}
}

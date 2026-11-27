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
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
	require.NoError(t, err)

	refreshed := strings.Repeat("b", accessTokenSize)
	arg, err := NewProfileWorkspaceOAuthArgument("https://keyring-refresh.cloud.databricks.test", profile)
	require.NoError(t, err)
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
	require.NoError(t, err)
	defer p.Close()

	tok, err := p.Token()
	require.NoError(t, err)
	assert.Equal(t, refreshed, tok.AccessToken)

	_, err = keyring.Get(service, profile+storage.AccessTokenPartMarker+"1")
	assert.NoError(t, err, "the refreshed token was not split")
	stored, err := storage.NewKeyringStore().Lookup(profile)
	require.NoError(t, err)
	assert.Equal(t, refreshed, stored.Token.AccessToken)
	assert.Equal(t, "second", stored.Token.RefreshToken)
}

//go:build darwin || windows

package auth

import (
	"os"
	"strings"
	"testing"
	"uuid"

	"github.com/databricks/cli/libs/auth/storage"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/databrickscfg/profile"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zalando/go-keyring"
	"golang.org/x/oauth2"
)

// TestLogoutRemovesSplitTokenFromOSKeyring logs out of a profile whose token
// the real OS keyring holds in several items, because it is too large for
// one, and checks that none of them is left. It uses the store that logout
// uses, with its 3-second timeout per keyring operation.
//
// It writes to the real OS keyring and therefore runs on CI only (GitHub
// Actions sets CI=true).
func TestLogoutRemovesSplitTokenFromOSKeyring(t *testing.T) {
	if os.Getenv("CI") == "" {
		t.Skip("touches the real OS keyring; runs on CI only")
	}

	// The keyring store keeps every item in this service.
	const service = "databricks-cli"
	profileName := "__probe_" + uuid.New().String()
	items := []string{
		profileName,
		profileName + storage.AccessTokenPartMarker + "0",
		profileName + storage.AccessTokenPartMarker + "1",
		profileName + storage.AccessTokenPartMarker + "2",
	}
	t.Cleanup(func() {
		for _, account := range items {
			_ = keyring.Delete(service, account)
		}
	})

	ctx := cmdio.MockDiscard(t.Context())
	configPath := writeTempConfig(t, "["+profileName+"]\nhost = https://keyring-logout.cloud.databricks.test\nauth_type = databricks-cli\n")
	t.Setenv("DATABRICKS_CONFIG_FILE", configPath)

	tokenStore := storage.NewKeyringStore()
	// The size of the access tokens in https://github.com/databricks/cli/issues/7707.
	tok := &oauth2.Token{AccessToken: strings.Repeat("a", 3250), RefreshToken: "refresh-token"}
	require.NoError(t, tokenStore.Put(profileName, storage.Entry{Token: tok}))
	_, err := keyring.Get(service, items[1])
	require.NoError(t, err, "the token was not split")

	err = runLogout(ctx, logoutArgs{
		profileName:    profileName,
		autoApprove:    true,
		profiler:       profile.DefaultProfiler,
		tokenStore:     tokenStore,
		configFilePath: configPath,
	})
	require.NoError(t, err)

	for _, account := range items {
		_, err := keyring.Get(service, account)
		assert.ErrorIs(t, err, keyring.ErrNotFound, "item %q is left in the OS keyring", account)
	}
}

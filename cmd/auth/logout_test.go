package auth

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/databrickscfg/profile"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2"
)

const logoutTestConfig = `[DEFAULT]
[my-workspace]
host = https://my-workspace.cloud.databricks.com

[shared-workspace]
host = https://my-workspace.cloud.databricks.com

[my-unique-workspace]
host = https://my-unique-workspace.cloud.databricks.com

[my-account]
host = https://accounts.cloud.databricks.com
account_id = abc123

[my-unified]
host = https://unified.cloud.databricks.com
account_id = def456
experimental_is_unified_host = true
`

var logoutTestTokensCacheConfig = map[string]*oauth2.Token{
	"my-workspace":        {AccessToken: "shared-workspace-token"},
	"shared-workspace":    {AccessToken: "shared-workspace-token"},
	"my-unique-workspace": {AccessToken: "my-unique-workspace-token"},
	"my-account":          {AccessToken: "my-account-token"},
	"my-unified":          {AccessToken: "my-unified-token"},
	"https://my-workspace.cloud.databricks.com":                  {AccessToken: "shared-workspace-host-token"},
	"https://my-unique-workspace.cloud.databricks.com":           {AccessToken: "unique-workspace-host-token"},
	"https://accounts.cloud.databricks.com/oidc/accounts/abc123": {AccessToken: "account-host-token"},
	"https://unified.cloud.databricks.com/oidc/accounts/def456":  {AccessToken: "unified-host-token"},
}

func writeTempConfig(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), ".databrickscfg")
	require.NoError(t, os.WriteFile(path, []byte(content), 0o600))
	return path
}

func TestLogout(t *testing.T) {
	cases := []struct {
		name         string
		profileName  string
		hostBasedKey string
		isSharedKey  bool
		force        bool
		wantErr      string
	}{
		{
			name:         "existing workspace profile with shared host",
			profileName:  "my-workspace",
			hostBasedKey: "https://my-workspace.cloud.databricks.com",
			isSharedKey:  true,
			force:        true,
		},
		{
			name:         "existing workspace profile with unique host",
			profileName:  "my-unique-workspace",
			hostBasedKey: "https://my-unique-workspace.cloud.databricks.com",
			isSharedKey:  false,
			force:        true,
		},
		{
			name:         "existing account profile",
			profileName:  "my-account",
			hostBasedKey: "https://accounts.cloud.databricks.com/oidc/accounts/abc123",
			isSharedKey:  false,
			force:        true,
		},
		{
			name:         "existing unified profile",
			profileName:  "my-unified",
			hostBasedKey: "https://unified.cloud.databricks.com/oidc/accounts/def456",
			isSharedKey:  false,
			force:        true,
		},
		{
			name:        "existing workspace profile without force in non-interactive mode",
			profileName: "my-workspace",
			force:       false,
			wantErr:     "please specify --force to skip confirmation in non-interactive mode",
		},
		{
			name:        "non-existing workspace profile",
			profileName: "nonexistent",
			force:       false,
			wantErr:     `profile "nonexistent" not found`,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := cmdio.MockDiscard(context.Background())
			configPath := writeTempConfig(t, logoutTestConfig)
			t.Setenv("DATABRICKS_CONFIG_FILE", configPath)

			tokenCache := &inMemoryTokenCache{
				Tokens: logoutTestTokensCacheConfig,
			}

			err := runLogout(ctx, logoutArgs{
				profileName:    tc.profileName,
				force:          tc.force,
				profiler:       profile.DefaultProfiler,
				tokenCache:     tokenCache,
				configFilePath: configPath,
			})

			if tc.wantErr != "" {
				assert.ErrorContains(t, err, tc.wantErr)
				return
			}
			require.NoError(t, err)

			// Verify profile was removed from config.
			profiles, err := profile.DefaultProfiler.LoadProfiles(ctx, profile.WithName(tc.profileName))
			require.NoError(t, err)
			assert.Empty(t, profiles, "expected profile %q to be removed", tc.profileName)

			// Verify tokens were cleaned up.
			assert.Nil(t, tokenCache.Tokens[tc.profileName], "expected token %q to be removed", tc.profileName)
			if tc.isSharedKey {
				assert.NotNil(t, tokenCache.Tokens[tc.hostBasedKey], "expected token %q to be preserved", tc.hostBasedKey)
			} else {
				assert.Nil(t, tokenCache.Tokens[tc.hostBasedKey], "expected token %q to be removed", tc.hostBasedKey)
			}
		})
	}
}

func TestLogoutNoTokens(t *testing.T) {
	ctx := cmdio.MockDiscard(context.Background())
	configPath := writeTempConfig(t, logoutTestConfig)
	t.Setenv("DATABRICKS_CONFIG_FILE", configPath)

	tokenCache := &inMemoryTokenCache{
		Tokens: map[string]*oauth2.Token{},
	}

	err := runLogout(ctx, logoutArgs{
		profileName:    "my-workspace",
		force:          true,
		profiler:       profile.DefaultProfiler,
		tokenCache:     tokenCache,
		configFilePath: configPath,
	})
	require.NoError(t, err)

	// Profile should still be removed from config even without cached tokens.
	profiles, err := profile.DefaultProfiler.LoadProfiles(ctx, profile.WithName("my-workspace"))
	require.NoError(t, err)
	assert.Empty(t, profiles)
}

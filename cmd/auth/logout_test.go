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

[staging]
host = https://staging.cloud.databricks.com

[shared-host-1]
host = https://shared.cloud.databricks.com

[shared-host-2]
host = https://shared.cloud.databricks.com
`

func writeTempConfig(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), ".databrickscfg")
	require.NoError(t, os.WriteFile(path, []byte(content), 0o600))
	return path
}

func TestLogout(t *testing.T) {
	cases := []struct {
		name        string
		profileName string
		force       bool
		wantErr     string
	}{
		{
			name:        "existing profile with force",
			profileName: "my-workspace",
			force:       true,
		},
		{
			name:        "existing profile without force in non-interactive mode",
			profileName: "my-workspace",
			force:       false,
			wantErr:     "please specify --force to skip confirmation in non-interactive mode",
		},
		{
			name:        "non-existing profile with force",
			profileName: "nonexistent",
			force:       true,
			wantErr:     `profile "nonexistent" not found`,
		},
		{
			name:        "non-existing profile without force",
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
				Tokens: map[string]*oauth2.Token{
					"my-workspace": {AccessToken: "token1"},
					"https://my-workspace.cloud.databricks.com": {AccessToken: "token1"},
				},
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
			assert.Empty(t, profiles)

			// Verify tokens were cleaned up.
			assert.Nil(t, tokenCache.Tokens["my-workspace"])
		})
	}
}

func TestLogoutSharedHost(t *testing.T) {
	ctx := cmdio.MockDiscard(context.Background())
	configPath := writeTempConfig(t, logoutTestConfig)
	t.Setenv("DATABRICKS_CONFIG_FILE", configPath)

	tokenCache := &inMemoryTokenCache{
		Tokens: map[string]*oauth2.Token{
			"shared-host-1":                             {AccessToken: "token1"},
			"shared-host-2":                             {AccessToken: "token2"},
			"https://shared.cloud.databricks.com":       {AccessToken: "shared-token"},
			"https://staging.cloud.databricks.com":      {AccessToken: "staging-token"},
			"https://my-workspace.cloud.databricks.com": {AccessToken: "ws-token"},
		},
	}

	err := runLogout(ctx, logoutArgs{
		profileName:    "shared-host-1",
		force:          true,
		profiler:       profile.DefaultProfiler,
		tokenCache:     tokenCache,
		configFilePath: configPath,
	})
	require.NoError(t, err)

	// Profile-keyed token should be removed.
	assert.Nil(t, tokenCache.Tokens["shared-host-1"])

	// Host-keyed token should be preserved because shared-host-2 still uses it.
	assert.NotNil(t, tokenCache.Tokens["https://shared.cloud.databricks.com"])

	// Other profiles' tokens should be untouched.
	assert.NotNil(t, tokenCache.Tokens["shared-host-2"])
	assert.NotNil(t, tokenCache.Tokens["https://staging.cloud.databricks.com"])
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

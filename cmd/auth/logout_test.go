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
			assert.Nil(t, tokenCache.Tokens[tc.profileName])
		})
	}
}

func TestLogoutTokenCacheCleanup(t *testing.T) {
	cases := []struct {
		name          string
		profileName   string
		tokens        map[string]*oauth2.Token
		wantRemoved   []string
		wantPreserved []string
	}{
		{
			name:        "workspace shared host preserves host-keyed token",
			profileName: "my-workspace",
			tokens: map[string]*oauth2.Token{
				"my-workspace":     {AccessToken: "token1"},
				"shared-workspace": {AccessToken: "token2"},
				"https://my-workspace.cloud.databricks.com": {AccessToken: "host-token"},
			},
			wantRemoved:   []string{"my-workspace"},
			wantPreserved: []string{"https://my-workspace.cloud.databricks.com", "shared-workspace"},
		},
		{
			name:        "workspace unique host clears host-keyed token",
			profileName: "my-unique-workspace",
			tokens: map[string]*oauth2.Token{
				"my-unique-workspace":                              {AccessToken: "token1"},
				"https://my-unique-workspace.cloud.databricks.com": {AccessToken: "host-token"},
			},
			wantRemoved: []string{"my-unique-workspace", "https://my-unique-workspace.cloud.databricks.com"},
		},
		{
			name:        "account profile clears OIDC-keyed token",
			profileName: "my-account",
			tokens: map[string]*oauth2.Token{
				"my-account": {AccessToken: "token1"},
				"https://accounts.cloud.databricks.com/oidc/accounts/abc123": {AccessToken: "account-token"},
			},
			wantRemoved: []string{"my-account", "https://accounts.cloud.databricks.com/oidc/accounts/abc123"},
		},
		{
			name:        "unified profile clears OIDC-keyed token",
			profileName: "my-unified",
			tokens: map[string]*oauth2.Token{
				"my-unified": {AccessToken: "token1"},
				"https://unified.cloud.databricks.com/oidc/accounts/def456": {AccessToken: "unified-token"},
			},
			wantRemoved: []string{"my-unified", "https://unified.cloud.databricks.com/oidc/accounts/def456"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := cmdio.MockDiscard(context.Background())
			configPath := writeTempConfig(t, logoutTestConfig)
			t.Setenv("DATABRICKS_CONFIG_FILE", configPath)

			tokenCache := &inMemoryTokenCache{Tokens: tc.tokens}

			err := runLogout(ctx, logoutArgs{
				profileName:    tc.profileName,
				force:          true,
				profiler:       profile.DefaultProfiler,
				tokenCache:     tokenCache,
				configFilePath: configPath,
			})
			require.NoError(t, err)

			for _, key := range tc.wantRemoved {
				assert.Nil(t, tokenCache.Tokens[key], "expected token %q to be removed", key)
			}
			for _, key := range tc.wantPreserved {
				assert.NotNil(t, tokenCache.Tokens[key], "expected token %q to be preserved", key)
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

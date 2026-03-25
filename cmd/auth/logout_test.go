package auth

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/databrickscfg"
	"github.com/databricks/cli/libs/databrickscfg/profile"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2"
)

const logoutTestConfig = `[DEFAULT]
[my-workspace]
host = https://my-workspace.cloud.databricks.com
auth_type  = databricks-cli

[shared-workspace]
host = https://my-workspace.cloud.databricks.com
auth_type  = databricks-cli

[my-unique-workspace]
host = https://my-unique-workspace.cloud.databricks.com
auth_type  = databricks-cli

[my-account]
host = https://accounts.cloud.databricks.com
account_id = abc123
auth_type  = databricks-cli

[my-unified]
host = https://unified.cloud.databricks.com
account_id = def456
experimental_is_unified_host = true
auth_type  = databricks-cli

[my-m2m]
host = https://my-m2m.cloud.databricks.com
token = dev-token
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
	"my-m2m":                              {AccessToken: "m2m-service-token"},
	"https://my-m2m.cloud.databricks.com": {AccessToken: "m2m-host-token"},
}

func copyTokens(src map[string]*oauth2.Token) map[string]*oauth2.Token {
	dst := make(map[string]*oauth2.Token, len(src))
	for k, v := range src {
		dst[k] = v
	}
	return dst
}

func writeTempConfig(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), ".databrickscfg")
	require.NoError(t, os.WriteFile(path, []byte(content), 0o600))
	return path
}

func TestLogout(t *testing.T) {
	cases := []struct {
		name          string
		profileName   string
		hostBasedKey  string
		isSharedKey   bool
		isNonU2M      bool // true for profiles that are not created by login (PAT, M2M, etc.)
		force         bool
		deleteProfile bool
		wantErr       string
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
		{
			name:          "delete workspace profile with shared host",
			profileName:   "my-workspace",
			hostBasedKey:  "https://my-workspace.cloud.databricks.com",
			isSharedKey:   true,
			force:         true,
			deleteProfile: true,
		},
		{
			name:          "delete workspace profile with unique host",
			profileName:   "my-unique-workspace",
			hostBasedKey:  "https://my-unique-workspace.cloud.databricks.com",
			isSharedKey:   false,
			force:         true,
			deleteProfile: true,
		},
		{
			name:          "delete account profile",
			profileName:   "my-account",
			hostBasedKey:  "https://accounts.cloud.databricks.com/oidc/accounts/abc123",
			isSharedKey:   false,
			force:         true,
			deleteProfile: true,
		},
		{
			name:          "delete unified profile",
			profileName:   "my-unified",
			hostBasedKey:  "https://unified.cloud.databricks.com/oidc/accounts/def456",
			isSharedKey:   false,
			force:         true,
			deleteProfile: true,
		},
		{
			name:          "do not delete m2m profile tokens",
			profileName:   "my-m2m",
			hostBasedKey:  "https://my-m2m.cloud.databricks.com",
			isNonU2M:      true,
			force:         true,
			deleteProfile: false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := cmdio.MockDiscard(t.Context())
			configPath := writeTempConfig(t, logoutTestConfig)
			t.Setenv("DATABRICKS_CONFIG_FILE", configPath)

			tokenCache := &inMemoryTokenCache{
				Tokens: copyTokens(logoutTestTokensCacheConfig),
			}

			err := runLogout(ctx, logoutArgs{
				profileName:    tc.profileName,
				force:          tc.force,
				deleteProfile:  tc.deleteProfile,
				profiler:       profile.DefaultProfiler,
				tokenCache:     tokenCache,
				configFilePath: configPath,
			})

			if tc.wantErr != "" {
				assert.ErrorContains(t, err, tc.wantErr)
				return
			}
			require.NoError(t, err)

			profiles, err := profile.DefaultProfiler.LoadProfiles(ctx, profile.WithName(tc.profileName))
			require.NoError(t, err)
			if tc.deleteProfile {
				assert.Empty(t, profiles, "expected profile %q to be removed", tc.profileName)
			} else {
				assert.NotEmpty(t, profiles, "expected profile %q to still exist", tc.profileName)
			}

			// Verify token cache state.
			if tc.isNonU2M {
				// Non-U2M profiles should not touch the token cache at all.
				assert.NotNil(t, tokenCache.Tokens[tc.profileName], "expected token %q to be preserved for non-U2M profile", tc.profileName)
				assert.NotNil(t, tokenCache.Tokens[tc.hostBasedKey], "expected token %q to be preserved for non-U2M profile", tc.hostBasedKey)
			} else {
				assert.Nil(t, tokenCache.Tokens[tc.profileName], "expected token %q to be removed", tc.profileName)
				if tc.isSharedKey {
					assert.NotNil(t, tokenCache.Tokens[tc.hostBasedKey], "expected token %q to be preserved", tc.hostBasedKey)
				} else {
					assert.Nil(t, tokenCache.Tokens[tc.hostBasedKey], "expected token %q to be removed", tc.hostBasedKey)
				}
			}
		})
	}
}

func TestLogoutNoTokens(t *testing.T) {
	ctx := cmdio.MockDiscard(t.Context())
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

	// Without --delete, profile should still exist.
	profiles, err := profile.DefaultProfiler.LoadProfiles(ctx, profile.WithName("my-workspace"))
	require.NoError(t, err)
	assert.NotEmpty(t, profiles)
}

func TestLogoutNoTokensWithDelete(t *testing.T) {
	ctx := cmdio.MockDiscard(t.Context())
	configPath := writeTempConfig(t, logoutTestConfig)
	t.Setenv("DATABRICKS_CONFIG_FILE", configPath)

	tokenCache := &inMemoryTokenCache{
		Tokens: map[string]*oauth2.Token{},
	}

	err := runLogout(ctx, logoutArgs{
		profileName:    "my-workspace",
		force:          true,
		deleteProfile:  true,
		profiler:       profile.DefaultProfiler,
		tokenCache:     tokenCache,
		configFilePath: configPath,
	})
	require.NoError(t, err)

	profiles, err := profile.DefaultProfiler.LoadProfiles(ctx, profile.WithName("my-workspace"))
	require.NoError(t, err)
	assert.Empty(t, profiles)
}

func TestResolveHostToProfileMatchesOneProfile(t *testing.T) {
	ctx := cmdio.MockDiscard(t.Context())
	profiler := profile.InMemoryProfiler{
		Profiles: profile.Profiles{
			{Name: "dev", Host: "https://dev.cloud.databricks.com", AuthType: "databricks-cli"},
			{Name: "staging", Host: "https://staging.cloud.databricks.com", AuthType: "databricks-cli"},
		},
	}

	resolved, err := resolveHostToProfile(ctx, "https://dev.cloud.databricks.com", profiler)
	require.NoError(t, err)
	assert.Equal(t, "dev", resolved)
}

func TestResolveHostToProfileMatchesMultipleProfiles(t *testing.T) {
	ctx := cmdio.MockDiscard(t.Context())
	profiler := profile.InMemoryProfiler{
		Profiles: profile.Profiles{
			{Name: "dev1", Host: "https://shared.cloud.databricks.com", AuthType: "databricks-cli"},
			{Name: "dev2", Host: "https://shared.cloud.databricks.com", AuthType: "databricks-cli"},
		},
	}

	_, err := resolveHostToProfile(ctx, "https://shared.cloud.databricks.com", profiler)
	assert.ErrorContains(t, err, "multiple profiles found matching host")
	assert.ErrorContains(t, err, "dev1")
	assert.ErrorContains(t, err, "dev2")
}

func TestResolveHostToProfileMatchesNothing(t *testing.T) {
	ctx := cmdio.MockDiscard(t.Context())
	profiler := profile.InMemoryProfiler{
		Profiles: profile.Profiles{
			{Name: "dev", Host: "https://dev.cloud.databricks.com", AuthType: "databricks-cli"},
			{Name: "staging", Host: "https://staging.cloud.databricks.com", AuthType: "databricks-cli"},
		},
	}

	_, err := resolveHostToProfile(ctx, "https://unknown.cloud.databricks.com", profiler)
	assert.ErrorContains(t, err, `no profile found matching host "https://unknown.cloud.databricks.com"`)
	assert.ErrorContains(t, err, "dev")
	assert.ErrorContains(t, err, "staging")
}

func TestResolveHostToProfileCanonicalizesHost(t *testing.T) {
	profiler := profile.InMemoryProfiler{
		Profiles: profile.Profiles{
			{Name: "dev", Host: "https://dev.cloud.databricks.com", AuthType: "databricks-cli"},
		},
	}

	cases := []struct {
		name string
		arg  string
	}{
		{name: "canonical URL", arg: "https://dev.cloud.databricks.com"},
		{name: "trailing slash", arg: "https://dev.cloud.databricks.com/"},
		{name: "no scheme", arg: "dev.cloud.databricks.com"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := cmdio.MockDiscard(t.Context())
			resolved, err := resolveHostToProfile(ctx, tc.arg, profiler)
			require.NoError(t, err)
			assert.Equal(t, "dev", resolved)
		})
	}
}

func TestLogoutProfileFlagAndPositionalArgConflict(t *testing.T) {
	parent := &cobra.Command{Use: "root"}
	parent.PersistentFlags().StringP("profile", "p", "", "~/.databrickscfg profile")
	cmd := newLogoutCommand()
	parent.AddCommand(cmd)
	parent.SetArgs([]string{"logout", "myprofile", "--profile", "other"})
	err := parent.Execute()
	assert.ErrorContains(t, err, "providing both --profile and a positional argument is not supported")
}

func TestLogoutDeleteClearsDefaultProfile(t *testing.T) {
	configWithDefault := `[DEFAULT]
[my-workspace]
host = https://my-workspace.cloud.databricks.com
auth_type  = databricks-cli

[other-workspace]
host = https://other-workspace.cloud.databricks.com
auth_type  = databricks-cli

[__settings__]
default_profile = my-workspace
`
	cases := []struct {
		name        string
		profileName string
		wantDefault string
	}{
		{
			name:        "deleting default profile clears default",
			profileName: "my-workspace",
			wantDefault: "",
		},
		{
			name:        "deleting non-default profile preserves default",
			profileName: "other-workspace",
			wantDefault: "my-workspace",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := cmdio.MockDiscard(t.Context())
			configPath := writeTempConfig(t, configWithDefault)
			t.Setenv("DATABRICKS_CONFIG_FILE", configPath)

			tokenCache := &inMemoryTokenCache{
				Tokens: copyTokens(logoutTestTokensCacheConfig),
			}

			err := runLogout(ctx, logoutArgs{
				profileName:    tc.profileName,
				force:          true,
				deleteProfile:  true,
				profiler:       profile.DefaultProfiler,
				tokenCache:     tokenCache,
				configFilePath: configPath,
			})
			require.NoError(t, err)

			got, err := databrickscfg.GetConfiguredDefaultProfile(ctx, configPath)
			require.NoError(t, err)
			assert.Equal(t, tc.wantDefault, got)
		})
	}
}

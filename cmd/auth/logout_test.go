package auth

import (
	"context"
	"encoding/json"
	"errors"
	"maps"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/databricks/cli/libs/auth"
	"github.com/databricks/cli/libs/auth/storage"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/databrickscfg"
	"github.com/databricks/cli/libs/databrickscfg/profile"
	"github.com/databricks/databricks-sdk-go/config"
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

[my-workspace-stale-account]
host = https://stale-account.cloud.databricks.com
account_id = stale-account
auth_type  = databricks-cli

[my-account]
host = https://accounts.cloud.databricks.com
account_id = abc123
auth_type  = databricks-cli

[my-m2m]
host = https://my-m2m.cloud.databricks.com
token = dev-token
`

var logoutTestTokensCacheConfig = map[string]*oauth2.Token{
	"my-workspace":                                               {AccessToken: "shared-workspace-token"},
	"shared-workspace":                                           {AccessToken: "shared-workspace-token"},
	"my-unique-workspace":                                        {AccessToken: "my-unique-workspace-token"},
	"my-workspace-stale-account":                                 {AccessToken: "stale-account-token"},
	"my-account":                                                 {AccessToken: "my-account-token"},
	"https://my-workspace.cloud.databricks.com":                  {AccessToken: "shared-workspace-host-token"},
	"https://my-unique-workspace.cloud.databricks.com":           {AccessToken: "unique-workspace-host-token"},
	"https://stale-account.cloud.databricks.com":                 {AccessToken: "stale-account-host-token"},
	"https://accounts.cloud.databricks.com/oidc/accounts/abc123": {AccessToken: "account-host-token"},
	"my-m2m":                              {AccessToken: "m2m-service-token"},
	"https://my-m2m.cloud.databricks.com": {AccessToken: "m2m-host-token"},
}

func copyTokens(src map[string]*oauth2.Token) map[string]*oauth2.Token {
	dst := make(map[string]*oauth2.Token, len(src))
	maps.Copy(dst, src)
	return dst
}

func writeTempConfig(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), ".databrickscfg")
	require.NoError(t, os.WriteFile(path, []byte(content), 0o600))
	return path
}

func stubFixtureHostMetadata(t *testing.T, endpoints map[string]string) {
	t.Helper()
	previous := config.DefaultHostMetadataResolverFactory
	config.DefaultHostMetadataResolverFactory = func(*config.Config) config.HostMetadataResolver {
		return func(_ context.Context, host string) (*config.HostMetadata, error) {
			endpoint, ok := endpoints[host]
			if !ok {
				t.Error("unexpected fixture metadata host")
				return nil, errors.New("unexpected fixture metadata host")
			}
			if endpoint == "" {
				return nil, nil
			}
			return &config.HostMetadata{OIDCEndpoint: endpoint}, nil
		}
	}
	t.Cleanup(func() { config.DefaultHostMetadataResolverFactory = previous })
}

// These cases test input selection, not a metadata response or HTTP transport.
func stubUnavailableFixtureMetadata(t *testing.T, inputs []string) {
	t.Helper()
	declared := make(map[string]string, len(inputs))
	for _, input := range inputs {
		copy := auth.AuthArguments{Host: input}
		require.NoError(t, normalizeLoginHost(&copy))
		cfg := &config.Config{Host: copy.Host}
		declared[cfg.CanonicalHostName()] = ""
	}
	stubFixtureHostMetadata(t, declared)
}

func stubLogoutHostMetadata(t *testing.T) {
	t.Helper()
	stubFixtureHostMetadata(t, map[string]string{
		"https://my-workspace.cloud.databricks.com":        "https://my-workspace.cloud.databricks.com/oidc",
		"https://my-unique-workspace.cloud.databricks.com": "https://my-unique-workspace.cloud.databricks.com/oidc",
		"https://stale-account.cloud.databricks.com":       "https://stale-account.cloud.databricks.com/oidc",
		"https://accounts.cloud.databricks.com":            "https://accounts.cloud.databricks.com/oidc/accounts/{account_id}",
		"https://my-m2m.cloud.databricks.com":              "https://my-m2m.cloud.databricks.com/oidc",
	})
}

func TestLogout(t *testing.T) {
	cases := []struct {
		name          string
		profileName   string
		hostBasedKey  string
		isSharedKey   bool
		isNonU2M      bool // true for profiles that are not created by login (PAT, M2M, etc.)
		autoApprove   bool
		deleteProfile bool
		wantErr       string
	}{
		{
			name:         "existing workspace profile with shared host",
			profileName:  "my-workspace",
			hostBasedKey: "https://my-workspace.cloud.databricks.com",
			isSharedKey:  true,
			autoApprove:  true,
		},
		{
			name:         "existing workspace profile with unique host",
			profileName:  "my-unique-workspace",
			hostBasedKey: "https://my-unique-workspace.cloud.databricks.com",
			isSharedKey:  false,
			autoApprove:  true,
		},
		{
			name:         "existing workspace profile with stale account id",
			profileName:  "my-workspace-stale-account",
			hostBasedKey: "https://stale-account.cloud.databricks.com",
			isSharedKey:  false,
			autoApprove:  true,
		},
		{
			name:         "existing account profile",
			profileName:  "my-account",
			hostBasedKey: "https://accounts.cloud.databricks.com/oidc/accounts/abc123",
			isSharedKey:  false,
			autoApprove:  true,
		},
		{
			name:        "existing workspace profile without auto-approve in non-interactive mode",
			profileName: "my-workspace",
			autoApprove: false,
			wantErr:     "please specify --auto-approve to skip confirmation in non-interactive mode",
		},
		{
			name:        "non-existing workspace profile",
			profileName: "nonexistent",
			autoApprove: false,
			wantErr:     `profile "nonexistent" not found`,
		},
		{
			name:          "delete workspace profile with shared host",
			profileName:   "my-workspace",
			hostBasedKey:  "https://my-workspace.cloud.databricks.com",
			isSharedKey:   true,
			autoApprove:   true,
			deleteProfile: true,
		},
		{
			name:          "delete workspace profile with unique host",
			profileName:   "my-unique-workspace",
			hostBasedKey:  "https://my-unique-workspace.cloud.databricks.com",
			isSharedKey:   false,
			autoApprove:   true,
			deleteProfile: true,
		},
		{
			name:          "delete account profile",
			profileName:   "my-account",
			hostBasedKey:  "https://accounts.cloud.databricks.com/oidc/accounts/abc123",
			isSharedKey:   false,
			autoApprove:   true,
			deleteProfile: true,
		},
		{
			name:          "do not delete m2m profile tokens",
			profileName:   "my-m2m",
			hostBasedKey:  "https://my-m2m.cloud.databricks.com",
			isNonU2M:      true,
			autoApprove:   true,
			deleteProfile: false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := cmdio.MockDiscard(t.Context())
			stubLogoutHostMetadata(t)
			configPath := writeTempConfig(t, logoutTestConfig)
			t.Setenv("DATABRICKS_CONFIG_FILE", configPath)

			tokenStore := &inMemoryStore{
				Tokens: copyTokens(logoutTestTokensCacheConfig),
			}

			err := runLogout(ctx, logoutArgs{
				profileName:    tc.profileName,
				autoApprove:    tc.autoApprove,
				deleteProfile:  tc.deleteProfile,
				profiler:       profile.DefaultProfiler,
				tokenStore:     tokenStore,
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
				assert.NotNil(t, tokenStore.Tokens[tc.profileName], "expected token %q to be preserved for non-U2M profile", tc.profileName)
				assert.NotNil(t, tokenStore.Tokens[tc.hostBasedKey], "expected token %q to be preserved for non-U2M profile", tc.hostBasedKey)
			} else {
				assert.Nil(t, tokenStore.Tokens[tc.profileName], "expected token %q to be removed", tc.profileName)
				if tc.isSharedKey {
					assert.NotNil(t, tokenStore.Tokens[tc.hostBasedKey], "expected token %q to be preserved", tc.hostBasedKey)
				} else {
					assert.Nil(t, tokenStore.Tokens[tc.hostBasedKey], "expected token %q to be removed", tc.hostBasedKey)
				}
			}
		})
	}
}

func TestLogoutNoTokens(t *testing.T) {
	ctx := cmdio.MockDiscard(t.Context())
	stubLogoutHostMetadata(t)
	configPath := writeTempConfig(t, logoutTestConfig)
	t.Setenv("DATABRICKS_CONFIG_FILE", configPath)

	tokenStore := &inMemoryStore{
		Tokens: map[string]*oauth2.Token{},
	}

	err := runLogout(ctx, logoutArgs{
		profileName:    "my-workspace",
		autoApprove:    true,
		profiler:       profile.DefaultProfiler,
		tokenStore:     tokenStore,
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
	stubLogoutHostMetadata(t)
	configPath := writeTempConfig(t, logoutTestConfig)
	t.Setenv("DATABRICKS_CONFIG_FILE", configPath)

	tokenStore := &inMemoryStore{
		Tokens: map[string]*oauth2.Token{},
	}

	err := runLogout(ctx, logoutArgs{
		profileName:    "my-workspace",
		autoApprove:    true,
		deleteProfile:  true,
		profiler:       profile.DefaultProfiler,
		tokenStore:     tokenStore,
		configFilePath: configPath,
	})
	require.NoError(t, err)

	profiles, err := profile.DefaultProfiler.LoadProfiles(ctx, profile.WithName("my-workspace"))
	require.NoError(t, err)
	assert.Empty(t, profiles)
}

// failingDeleteStore fails every Delete, like a keyring that is unavailable.
type failingDeleteStore struct {
	*inMemoryStore
}

func (failingDeleteStore) Delete(string) error {
	return errors.New("keyring unavailable")
}

// A profile that auth login did not create does not use the token store, so
// an unavailable store does not fail its logout.
func TestLogoutNonLoginProfileIgnoresTokenStoreError(t *testing.T) {
	ctx := cmdio.MockDiscard(t.Context())
	stubLogoutHostMetadata(t)
	configPath := writeTempConfig(t, logoutTestConfig)
	t.Setenv("DATABRICKS_CONFIG_FILE", configPath)

	err := runLogout(ctx, logoutArgs{
		profileName:    "my-m2m",
		autoApprove:    true,
		deleteProfile:  true,
		profiler:       profile.DefaultProfiler,
		tokenStore:     failingDeleteStore{&inMemoryStore{Tokens: map[string]*oauth2.Token{}}},
		configFilePath: configPath,
	})
	require.NoError(t, err)

	profiles, err := profile.DefaultProfiler.LoadProfiles(ctx, profile.WithName("my-m2m"))
	require.NoError(t, err)
	assert.Empty(t, profiles)
}

func TestLogoutProfileFlagAndPositionalArgConflict(t *testing.T) {
	parent := &cobra.Command{Use: "root"}
	parent.PersistentFlags().StringP("profile", "p", "", "~/.databrickscfg profile")
	cmd := newLogoutCommand()
	parent.AddCommand(cmd)
	parent.SetArgs([]string{"logout", "myprofile", "--profile", "other"})
	err := parent.Execute()
	assert.ErrorContains(t, err, `argument "myprofile" cannot be combined with --profile`)
}

func TestLogoutDeleteClearsDefaultProfile(t *testing.T) {
	stubFixtureHostMetadata(t, map[string]string{
		"https://my-workspace.cloud.databricks.com":    "https://my-workspace.cloud.databricks.com/oidc",
		"https://other-workspace.cloud.databricks.com": "https://other-workspace.cloud.databricks.com/oidc",
	})
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

			tokenStore := &inMemoryStore{
				Tokens: copyTokens(logoutTestTokensCacheConfig),
			}

			err := runLogout(ctx, logoutArgs{
				profileName:    tc.profileName,
				autoApprove:    true,
				deleteProfile:  true,
				profiler:       profile.DefaultProfiler,
				tokenStore:     tokenStore,
				configFilePath: configPath,
			})
			require.NoError(t, err)

			got, err := databrickscfg.GetConfiguredDefaultProfile(ctx, configPath)
			require.NoError(t, err)
			assert.Equal(t, tc.wantDefault, got)
		})
	}
}

// newWellKnownServer creates a mock server that serves /.well-known/databricks-config
// with the given oidc_endpoint shape. Use accountScoped=true for SPOG hosts.
func newWellKnownServer(t *testing.T, accountScoped bool, accountID string) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/.well-known/databricks-config" {
			w.Header().Set("Content-Type", "application/json")
			oidcEndpoint := r.Host + "/oidc"
			if accountScoped {
				oidcEndpoint = r.Host + "/oidc/accounts/" + accountID
			}
			_ = json.NewEncoder(w).Encode(map[string]any{
				"account_id":    accountID,
				"oidc_endpoint": oidcEndpoint,
			})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	t.Cleanup(server.Close)
	return server
}

func TestLogoutSPOGProfile(t *testing.T) {
	spogServer := newWellKnownServer(t, true, "spog-acct")

	ctx := cmdio.MockDiscard(t.Context())
	configPath := writeTempConfig(t, `[DEFAULT]
[spog-profile]
host = `+spogServer.URL+`
account_id = spog-acct
workspace_id = spog-ws
auth_type = databricks-cli
`)
	t.Setenv("DATABRICKS_CONFIG_FILE", configPath)

	hostKey := spogServer.URL + "/oidc/accounts/spog-acct"
	tokenStore := &inMemoryStore{
		Tokens: map[string]*oauth2.Token{
			"spog-profile": {AccessToken: "spog-profile-token"},
			hostKey:        {AccessToken: "spog-host-token"},
		},
	}

	err := runLogout(ctx, logoutArgs{
		profileName:    "spog-profile",
		autoApprove:    true,
		profiler:       profile.DefaultProfiler,
		tokenStore:     tokenStore,
		configFilePath: configPath,
	})
	require.NoError(t, err)

	assert.Nil(t, tokenStore.Tokens["spog-profile"])
	assert.Nil(t, tokenStore.Tokens[hostKey])
}

func TestClearFileTokenStorePreservesOtherEntries(t *testing.T) {
	for _, shared := range []bool{false, true} {
		t.Run(map[bool]string{false: "unique-host", true: "shared-host"}[shared], func(t *testing.T) {
			server := newWellKnownServer(t, false, "synthetic-account")
			owner := profile.Profile{Name: "owner", Host: server.URL, AuthType: "databricks-cli"}
			profiles := profile.Profiles{owner}
			if shared {
				profiles = append(profiles, profile.Profile{Name: "sibling", Host: server.URL, AuthType: "databricks-cli"})
			}
			path := filepath.Join(t.TempDir(), "tokens.json")
			store, err := storage.NewFileStore(t.Context(), storage.WithFileLocation(path))
			require.NoError(t, err)
			for _, key := range []string{owner.Name, server.URL, "unrelated"} {
				require.NoError(t, store.Put(key, storage.Entry{Token: &oauth2.Token{AccessToken: "synthetic"}}))
				require.NoError(t, store.Put(pendingLoginKey(key), storage.Entry{PendingLogin: &storage.PendingLogin{UserCode: "synthetic"}}))
			}
			require.NoError(t, clearTokenStore(t.Context(), owner, profile.InMemoryProfiler{Profiles: profiles}, store))
			for _, key := range []string{owner.Name, pendingLoginKey(owner.Name)} {
				_, err := store.Lookup(key)
				assert.ErrorIs(t, err, storage.ErrNotFound)
			}
			for _, key := range []string{server.URL, pendingLoginKey(server.URL)} {
				_, err := store.Lookup(key)
				if shared {
					assert.NoError(t, err)
				} else {
					assert.ErrorIs(t, err, storage.ErrNotFound)
				}
			}
			for _, key := range []string{"unrelated", pendingLoginKey("unrelated")} {
				_, err := store.Lookup(key)
				assert.NoError(t, err)
			}
		})
	}
}

func TestHostCacheKeyAndMatchFn(t *testing.T) {
	wsServer := newWellKnownServer(t, false, "ws-account")
	spogServer := newWellKnownServer(t, true, "spog-account")

	cases := []struct {
		name         string
		profile      profile.Profile
		wantKey      string
		wantKeyEmpty bool
	}{
		{
			name: "classic workspace",
			profile: profile.Profile{
				Name: "ws",
				Host: wsServer.URL,
			},
			wantKey: wsServer.URL,
		},
		{
			name: "workspace with stale account_id",
			profile: profile.Profile{
				Name:      "stale",
				Host:      wsServer.URL,
				AccountID: "stale-account",
			},
			wantKey: wsServer.URL,
		},
		{
			name: "classic account host",
			profile: profile.Profile{
				Name:      "acct",
				Host:      "https://accounts.cloud.databricks.com",
				AccountID: "abc123",
			},
			wantKey: "https://accounts.cloud.databricks.com/oidc/accounts/abc123",
		},
		{
			name: "SPOG profile routes to account key via discovery",
			profile: profile.Profile{
				Name:      "spog",
				Host:      spogServer.URL,
				AccountID: "spog-account",
			},
			wantKey: spogServer.URL + "/oidc/accounts/spog-account",
		},
		{
			name: "empty host returns empty",
			profile: profile.Profile{
				Name: "no-host",
			},
			wantKeyEmpty: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.profile.Host == "https://accounts.cloud.databricks.com" {
				stubFixtureHostMetadata(t, map[string]string{
					tc.profile.Host: tc.profile.Host + "/oidc/accounts/{account_id}",
				})
			}
			key, matchFn := hostCacheKeyAndMatchFn(tc.profile)
			if tc.wantKeyEmpty {
				assert.Empty(t, key)
				assert.Nil(t, matchFn)
				return
			}
			assert.Equal(t, tc.wantKey, key)
			require.NotNil(t, matchFn)
		})
	}
}

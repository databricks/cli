package storage

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestResolveStorageMode covers the precedence rules of the pure core
// resolveStorageMode. No env, no files; inputs are plain strings.
func TestResolveStorageMode(t *testing.T) {
	cases := []struct {
		name        string
		override    StorageMode
		envValue    string
		configValue string
		want        StorageMode
		wantErrSub  string
	}{
		{
			name: "default when nothing is set",
			want: StorageModeLegacy,
		},
		{
			name:        "override wins over env and config",
			override:    StorageModeSecure,
			envValue:    "plaintext",
			configValue: "legacy",
			want:        StorageModeSecure,
		},
		{
			name:        "env wins over config",
			envValue:    "secure",
			configValue: "plaintext",
			want:        StorageModeSecure,
		},
		{
			name:        "config sets mode when env and override unset",
			configValue: "secure",
			want:        StorageModeSecure,
		},
		{
			name:     "env value is case-insensitive and trimmed",
			envValue: "  SECURE  ",
			want:     StorageModeSecure,
		},
		{
			name:       "invalid override is rejected",
			override:   StorageMode("bogus"),
			wantErrSub: `unknown storage mode "bogus"`,
		},
		{
			name:       "invalid env is rejected",
			envValue:   "bogus",
			wantErrSub: "DATABRICKS_AUTH_STORAGE",
		},
		{
			name:        "invalid config value is rejected",
			configValue: "bogus",
			wantErrSub:  "auth_storage",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := resolveStorageMode(tc.override, tc.envValue, tc.configValue)
			if tc.wantErrSub != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.wantErrSub)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.want, got)
		})
	}
}

// TestResolveStorageMode_ReadsEnvAndConfig exercises the I/O wrapper. The
// precedence rules are covered by TestResolveStorageMode above; these
// cases verify that the wrapper actually reads from the env and from
// [__settings__].auth_storage.
func TestResolveStorageMode_ReadsEnvAndConfig(t *testing.T) {
	t.Run("env value is picked up", func(t *testing.T) {
		cfgPath := filepath.Join(t.TempDir(), ".databrickscfg")
		require.NoError(t, os.WriteFile(cfgPath, []byte("[__settings__]\nauth_storage = legacy\n"), 0o600))
		t.Setenv("DATABRICKS_CONFIG_FILE", cfgPath)
		t.Setenv(EnvVar, "secure")

		got, err := ResolveStorageMode(t.Context(), "")
		require.NoError(t, err)
		assert.Equal(t, StorageModeSecure, got)
	})

	t.Run("config value is picked up when env unset", func(t *testing.T) {
		cfgPath := filepath.Join(t.TempDir(), ".databrickscfg")
		require.NoError(t, os.WriteFile(cfgPath, []byte("[__settings__]\nauth_storage = secure\n"), 0o600))
		t.Setenv("DATABRICKS_CONFIG_FILE", cfgPath)
		t.Setenv(EnvVar, "")

		got, err := ResolveStorageMode(t.Context(), "")
		require.NoError(t, err)
		assert.Equal(t, StorageModeSecure, got)
	})

	t.Run("defaults to legacy when both unset", func(t *testing.T) {
		t.Setenv("DATABRICKS_CONFIG_FILE", filepath.Join(t.TempDir(), "does-not-exist"))
		t.Setenv(EnvVar, "")

		got, err := ResolveStorageMode(t.Context(), "")
		require.NoError(t, err)
		assert.Equal(t, StorageModeLegacy, got)
	})
}

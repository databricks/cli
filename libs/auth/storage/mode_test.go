package storage

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseMode(t *testing.T) {
	cases := []struct {
		name string
		raw  string
		want StorageMode
	}{
		{name: "empty returns unknown", raw: "", want: StorageModeUnknown},
		{name: "whitespace returns unknown", raw: "   ", want: StorageModeUnknown},
		{name: "plaintext lowercase", raw: "plaintext", want: StorageModePlaintext},
		{name: "secure lowercase", raw: "secure", want: StorageModeSecure},
		{name: "case and whitespace normalized", raw: "  SECURE  ", want: StorageModeSecure},
		{name: "legacy keyword no longer recognized", raw: "legacy", want: StorageModeUnknown},
		{name: "unknown value returns unknown", raw: "bogus", want: StorageModeUnknown},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, ParseMode(tc.raw))
		})
	}
}

func TestResolveStorageMode(t *testing.T) {
	cases := []struct {
		name       string
		override   StorageMode
		envValue   string
		configBody string
		want       StorageMode
		wantErrSub string
	}{
		{
			name: "default when nothing is set",
			want: StorageModePlaintext,
		},
		{
			name:       "override wins over env and config",
			override:   StorageModeSecure,
			envValue:   "plaintext",
			configBody: "[__settings__]\nauth_storage = plaintext\n",
			want:       StorageModeSecure,
		},
		{
			name:     "override is trusted (not validated)",
			override: StorageMode("bogus"),
			want:     StorageMode("bogus"),
		},
		{
			name:       "env wins over config",
			envValue:   "secure",
			configBody: "[__settings__]\nauth_storage = plaintext\n",
			want:       StorageModeSecure,
		},
		{
			name:       "config sets mode when env and override unset",
			configBody: "[__settings__]\nauth_storage = secure\n",
			want:       StorageModeSecure,
		},
		{
			name:     "env value is case-insensitive and trimmed",
			envValue: "  SECURE  ",
			want:     StorageModeSecure,
		},
		{
			name:       "invalid env is rejected",
			envValue:   "bogus",
			wantErrSub: "DATABRICKS_AUTH_STORAGE",
		},
		{
			name:       "invalid config value is rejected",
			configBody: "[__settings__]\nauth_storage = bogus\n",
			wantErrSub: "auth_storage",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cfgPath := filepath.Join(t.TempDir(), ".databrickscfg")
			if tc.configBody != "" {
				require.NoError(t, os.WriteFile(cfgPath, []byte(tc.configBody), 0o600))
			}
			t.Setenv("DATABRICKS_CONFIG_FILE", cfgPath)
			t.Setenv(EnvVar, tc.envValue)

			got, err := ResolveStorageMode(t.Context(), tc.override)
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

// TestResolveStorageMode_SkipsConfigReadWhenOverrideOrEnvSet verifies that
// ResolveStorageMode short-circuits before reading .databrickscfg when an
// earlier source already decided the mode. A deliberately broken config path
// would produce an error if the read happened.
func TestResolveStorageMode_SkipsConfigReadWhenOverrideOrEnvSet(t *testing.T) {
	// Point DATABRICKS_CONFIG_FILE at a path that is not a regular file so
	// any attempted read surfaces as an error.
	unreadableDir := t.TempDir()
	t.Setenv("DATABRICKS_CONFIG_FILE", unreadableDir)

	t.Run("override short-circuits", func(t *testing.T) {
		t.Setenv(EnvVar, "")
		got, err := ResolveStorageMode(t.Context(), StorageModeSecure)
		require.NoError(t, err)
		assert.Equal(t, StorageModeSecure, got)
	})

	t.Run("env short-circuits", func(t *testing.T) {
		t.Setenv(EnvVar, "secure")
		got, err := ResolveStorageMode(t.Context(), StorageModeUnknown)
		require.NoError(t, err)
		assert.Equal(t, StorageModeSecure, got)
	})
}

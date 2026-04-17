package storage_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/databricks/cli/libs/auth/storage"
	"github.com/databricks/cli/libs/env"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolveStorageMode(t *testing.T) {
	cases := []struct {
		name       string
		override   storage.StorageMode
		envVal     string
		configBody string
		want       storage.StorageMode
		wantErrSub string
	}{
		{
			name: "default when nothing is set",
			want: storage.StorageModeLegacy,
		},
		{
			name:       "override wins over env and config",
			override:   storage.StorageModeSecure,
			envVal:     "plaintext",
			configBody: "[__settings__]\nauth_storage = legacy\n",
			want:       storage.StorageModeSecure,
		},
		{
			name:       "env wins over config",
			envVal:     "secure",
			configBody: "[__settings__]\nauth_storage = plaintext\n",
			want:       storage.StorageModeSecure,
		},
		{
			name:       "config sets mode when env and override unset",
			configBody: "[__settings__]\nauth_storage = secure\n",
			want:       storage.StorageModeSecure,
		},
		{
			name:   "env value is case-insensitive and trimmed",
			envVal: "  SECURE  ",
			want:   storage.StorageModeSecure,
		},
		{
			name:       "invalid override is rejected",
			override:   storage.StorageMode("bogus"),
			wantErrSub: `unknown storage mode "bogus"`,
		},
		{
			name:       "invalid env is rejected",
			envVal:     "bogus",
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
			ctx := t.Context()

			cfgPath := filepath.Join(t.TempDir(), ".databrickscfg")
			if tc.configBody != "" {
				require.NoError(t, os.WriteFile(cfgPath, []byte(tc.configBody), 0o600))
			}
			ctx = env.Set(ctx, "DATABRICKS_CONFIG_FILE", cfgPath)

			if tc.envVal != "" {
				ctx = env.Set(ctx, storage.EnvVar, tc.envVal)
			}

			got, err := storage.ResolveStorageMode(ctx, tc.override)
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

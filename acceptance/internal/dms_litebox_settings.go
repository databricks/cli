package internal

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/databricks/cli/libs/env"
	"github.com/stretchr/testify/require"
)

// ConfigureDMSLiteboxFromFile fills missing LITE settings from the local fixture's discovery file.
func ConfigureDMSLiteboxFromFile(t *testing.T) {
	t.Helper()
	ctx := t.Context()
	keys := []string{"DMS_LITEBOX_URL", "DMS_LITEBOX_CERT", "DMS_LITEBOX_KEY", "DMS_LITEBOX_CA"}
	missing := false
	for _, key := range keys {
		if env.Get(ctx, key) == "" {
			missing = true
		}
	}
	if !missing {
		return
	}

	acceptanceDir, err := os.Getwd()
	require.NoError(t, err)
	file := env.Get(ctx, "DMS_CLI_ENDPOINTS_FILE")
	explicit := file != ""
	if !explicit {
		file = findDMSLiteboxSettings(t, filepath.Dir(acceptanceDir))
		if file == "" {
			return
		}
	}
	if !filepath.IsAbs(file) {
		file = filepath.Join(acceptanceDir, file)
	}
	data, err := os.ReadFile(file)
	if !explicit && errors.Is(err, os.ErrNotExist) {
		return
	}
	require.NoError(t, err, "read DMS LITE settings %s", file)
	var settings struct {
		URL  string `json:"url"`
		Cert string `json:"cert"`
		Key  string `json:"key"`
		CA   string `json:"ca"`
	}
	require.NoError(t, json.Unmarshal(data, &settings), "parse DMS LITE settings %s", file)
	for _, setting := range []struct{ key, value string }{
		{"DMS_LITEBOX_URL", settings.URL},
		{"DMS_LITEBOX_CERT", settings.Cert},
		{"DMS_LITEBOX_KEY", settings.Key},
		{"DMS_LITEBOX_CA", settings.CA},
	} {
		if env.Get(ctx, setting.key) == "" && setting.value != "" {
			t.Setenv(setting.key, setting.value)
		}
	}
}

func findDMSLiteboxSettings(t *testing.T, dir string) string {
	t.Helper()
	for {
		file := filepath.Join(dir, "bundle", "metadata-service", "test", ".lite", "cli-endpoints.json")
		_, err := os.Stat(file)
		if err == nil {
			return file
		}
		require.ErrorIs(t, err, os.ErrNotExist, "find DMS LITE settings %s", file)
		parent := filepath.Dir(dir)
		if parent == dir {
			return ""
		}
		dir = parent
	}
}

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

type dmsLiteboxSettings struct {
	URL  string `json:"url"`
	Cert string `json:"cert"`
	Key  string `json:"key"`
	CA   string `json:"ca"`
}

func loadDMSLiteboxSettings(t *testing.T) *dmsLiteboxSettings {
	t.Helper()
	acceptanceDir, err := os.Getwd()
	require.NoError(t, err)
	file := env.Get(t.Context(), "DMS_CLI_ENDPOINTS_FILE")
	explicit := file != ""
	if !explicit {
		file = findDMSLiteboxSettings(t, filepath.Dir(acceptanceDir))
		if file == "" {
			return nil
		}
	}
	if !filepath.IsAbs(file) {
		file = filepath.Join(acceptanceDir, file)
	}
	data, err := os.ReadFile(file)
	if !explicit && errors.Is(err, os.ErrNotExist) {
		return nil
	}
	require.NoError(t, err, "read DMS LITE settings %s", file)
	var settings dmsLiteboxSettings
	require.NoError(t, json.Unmarshal(data, &settings), "parse DMS LITE settings %s", file)
	require.NotEmpty(t, settings.URL, "DMS LITE settings %s: url is required", file)
	require.NotEmpty(t, settings.Cert, "DMS LITE settings %s: cert is required", file)
	require.NotEmpty(t, settings.Key, "DMS LITE settings %s: key is required", file)
	require.NotEmpty(t, settings.CA, "DMS LITE settings %s: ca is required", file)
	return &settings
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

package internal

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/databricks/cli/libs/env"
	"github.com/databricks/cli/libs/folders"
	"github.com/stretchr/testify/require"
)

const dmsEndpointsFileEnv = "DMS_CLI_ENDPOINTS_FILE"

var dmsLiteboxSettingsKeys = []string{
	"DMS_LITEBOX_URL", "DMS_LITEBOX_CERT", "DMS_LITEBOX_KEY", "DMS_LITEBOX_CA",
}

// ConfigureDMSLiteboxFromFile fills missing LITE settings from the local fixture's discovery file.
func ConfigureDMSLiteboxFromFile(t *testing.T) {
	t.Helper()
	dir, err := os.Getwd()
	require.NoError(t, err)
	settings, err := loadDMSLiteboxSettings(t.Context(), dir)
	require.NoError(t, err)
	for _, key := range dmsLiteboxSettingsKeys {
		if env.Get(t.Context(), key) == "" && settings[key] != "" {
			t.Setenv(key, settings[key])
		}
	}
}

func loadDMSLiteboxSettings(ctx context.Context, dir string) (map[string]string, error) {
	settings := make(map[string]string, len(dmsLiteboxSettingsKeys))
	complete := true
	for _, key := range dmsLiteboxSettingsKeys {
		settings[key] = env.Get(ctx, key)
		complete = complete && settings[key] != ""
	}
	if complete {
		return settings, nil
	}

	file := env.Get(ctx, dmsEndpointsFileEnv)
	explicit := file != ""
	if !explicit {
		var err error
		file, err = findDMSLiteboxSettings(dir)
		if err != nil {
			return nil, err
		}
		if file == "" {
			return settings, nil
		}
	}
	if !filepath.IsAbs(file) {
		file = filepath.Join(dir, file)
	}
	data, err := os.ReadFile(file)
	if !explicit && errors.Is(err, os.ErrNotExist) {
		return settings, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read DMS LITE settings %s: %w", filepath.ToSlash(file), err)
	}
	var config struct {
		URL  string `json:"url"`
		Cert string `json:"cert"`
		Key  string `json:"key"`
		CA   string `json:"ca"`
	}
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("parse DMS LITE settings %s: %w", filepath.ToSlash(file), err)
	}
	defaults := []string{config.URL, config.Cert, config.Key, config.CA}
	for i, key := range dmsLiteboxSettingsKeys {
		if settings[key] == "" {
			settings[key] = defaults[i]
		}
	}
	return settings, nil
}

func findDMSLiteboxSettings(dir string) (string, error) {
	for {
		root, err := folders.FindDirWithLeaf(dir, ".git")
		if errors.Is(err, os.ErrNotExist) {
			return "", nil
		}
		if err != nil {
			return "", fmt.Errorf("find Git checkout for DMS LITE settings: %w", err)
		}
		fixture := filepath.Join(root, "bundle", "metadata-service", "test")
		info, err := os.Stat(fixture)
		if err == nil && info.IsDir() {
			return filepath.Join(fixture, ".lite", "cli-endpoints.json"), nil
		}
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			return "", fmt.Errorf("find DMS LITE fixture: %w", err)
		}
		dir = filepath.Dir(root)
		if dir == root {
			return "", nil
		}
	}
}

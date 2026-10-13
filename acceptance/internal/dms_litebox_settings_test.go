package internal

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/databricks/cli/libs/env"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func emptyDMSLiteboxSettingsContext(t *testing.T) context.Context {
	t.Helper()
	ctx := env.Set(t.Context(), dmsEndpointsFileEnv, "")
	for _, key := range dmsLiteboxSettingsKeys {
		ctx = env.Set(ctx, key, "")
	}
	return ctx
}

func writeDMSLiteboxSettings(t *testing.T, file string) {
	t.Helper()
	data, err := json.Marshal(map[string]string{
		"url": "https://127.0.0.1:12345", "workspace_url": "https://127.0.0.1:12346",
		"cert": "certificate.crt", "key": "certificate.key", "ca": "ca.crt",
	})
	require.NoError(t, err)
	require.NoError(t, os.MkdirAll(filepath.Dir(file), 0o755))
	require.NoError(t, os.WriteFile(file, data, 0o600))
}

func TestDMSLiteboxSettingsDiscovery(t *testing.T) {
	for _, nested := range []bool{false, true} {
		name := "inside Universe checkout"
		if nested {
			name = "inside arbitrarily located nested CLI checkout"
		}
		t.Run(name, func(t *testing.T) {
			root := t.TempDir()
			require.NoError(t, os.Mkdir(filepath.Join(root, ".git"), 0o755))
			file := filepath.Join(root, "bundle", "metadata-service", "test", ".lite", "cli-endpoints.json")
			writeDMSLiteboxSettings(t, file)
			dir := filepath.Join(root, "somewhere", "cli", "acceptance", "internal")
			require.NoError(t, os.MkdirAll(dir, 0o755))
			if nested {
				require.NoError(t, os.Mkdir(filepath.Join(root, "somewhere", "cli", ".git"), 0o755))
			}
			found, err := findDMSLiteboxSettings(dir)
			require.NoError(t, err)
			assert.Equal(t, file, found)
			settings, err := loadDMSLiteboxSettings(emptyDMSLiteboxSettingsContext(t), dir)
			require.NoError(t, err)
			assert.Equal(t, "https://127.0.0.1:12345", settings["DMS_LITEBOX_URL"])
			assert.Equal(t, "https://127.0.0.1:12346", settings["DMS_LITEBOX_WORKSPACE_URL"])
			assert.Equal(t, "certificate.crt", settings["DMS_LITEBOX_CERT"])
			assert.Equal(t, "certificate.key", settings["DMS_LITEBOX_KEY"])
			assert.Equal(t, "ca.crt", settings["DMS_LITEBOX_CA"])
		})
	}
}

func TestDMSLiteboxSettingsExplicitFileAndOverrides(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "custom", "settings.json")
	writeDMSLiteboxSettings(t, file)
	for _, path := range []string{file, filepath.Join("custom", "settings.json")} {
		ctx := env.Set(emptyDMSLiteboxSettingsContext(t), dmsEndpointsFileEnv, path)
		ctx = env.Set(ctx, "DMS_LITEBOX_URL", "https://127.0.0.1:54321")
		settings, err := loadDMSLiteboxSettings(ctx, dir)
		require.NoError(t, err)
		assert.Equal(t, "https://127.0.0.1:54321", settings["DMS_LITEBOX_URL"])
		assert.Equal(t, "https://127.0.0.1:12346", settings["DMS_LITEBOX_WORKSPACE_URL"])
		assert.Equal(t, "ca.crt", settings["DMS_LITEBOX_CA"])
	}
}

func TestDMSLiteboxSettingsAbsent(t *testing.T) {
	dir := t.TempDir()
	// A fixture-shaped directory without a Git checkout must not enable LITE.
	writeDMSLiteboxSettings(t, filepath.Join(dir, "bundle", "metadata-service", "test", ".lite", "cli-endpoints.json"))
	settings, err := loadDMSLiteboxSettings(emptyDMSLiteboxSettingsContext(t), dir)
	require.NoError(t, err)
	assert.Empty(t, settings["DMS_LITEBOX_URL"])
	require.NoError(t, os.Mkdir(filepath.Join(dir, ".git"), 0o755))
	require.NoError(t, os.Remove(filepath.Join(dir, "bundle", "metadata-service", "test", ".lite", "cli-endpoints.json")))
	settings, err = loadDMSLiteboxSettings(emptyDMSLiteboxSettingsContext(t), dir)
	require.NoError(t, err)
	assert.Empty(t, settings["DMS_LITEBOX_URL"])
}

func TestDMSLiteboxSettingsInvalidFile(t *testing.T) {
	for _, content := range []string{"missing", "{", `{"url":42}`} {
		t.Run(content, func(t *testing.T) {
			dir := t.TempDir()
			file := filepath.Join(dir, "settings.json")
			if content != "missing" {
				require.NoError(t, os.WriteFile(file, []byte(content), 0o600))
			}
			ctx := env.Set(emptyDMSLiteboxSettingsContext(t), dmsEndpointsFileEnv, file)
			_, err := loadDMSLiteboxSettings(ctx, dir)
			assert.ErrorContains(t, err, "DMS LITE settings")
		})
	}
}

func TestDMSLiteboxSettingsExplicitEnvironmentNeedsNoFile(t *testing.T) {
	ctx := env.Set(emptyDMSLiteboxSettingsContext(t), dmsEndpointsFileEnv, "missing.json")
	for _, key := range dmsLiteboxSettingsKeys {
		ctx = env.Set(ctx, key, "explicit")
	}
	settings, err := loadDMSLiteboxSettings(ctx, t.TempDir())
	require.NoError(t, err)
	for _, key := range dmsLiteboxSettingsKeys {
		assert.Equal(t, "explicit", settings[key])
	}
}

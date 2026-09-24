package acceptance_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildTestEnvAliasPrecedence(t *testing.T) {
	const full = "DATABRICKS_BUNDLE_DEPLOYMENT_HISTORY"
	tests := []struct {
		name   string
		config map[string]string
		custom []string
		want   []string
	}{
		{
			name:   "alias wins inherited full name",
			config: map[string]string{full: "inherited", "OTHER": "kept"},
			custom: []string{"DMS=true"},
			want:   []string{"OTHER=kept", "DMS=true", full + "=true"},
		},
		{
			name:   "explicit full name wins alias",
			custom: []string{"DMS=alias", full + "=full"},
			want:   []string{"DMS=alias", full + "=full"},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert.Equal(t, test.want, buildTestEnv(test.config, test.custom))
		})
	}
}

func TestCopyDirExcludesRootTestToml(t *testing.T) {
	src := t.TempDir()
	dst := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(src, "test.toml"), []byte("Cloud = true\n"), 0o644))
	require.NoError(t, os.MkdirAll(filepath.Join(src, "nested"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(src, "nested", "test.toml"), []byte("Cloud = false\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(src, "databricks.yml"), []byte("bundle: {}\n"), 0o644))

	inputs := map[string]bool{}
	outputs := map[string]bool{}
	require.NoError(t, CopyDir(src, dst, inputs, outputs))
	assert.False(t, inputs["test.toml"])
	assert.True(t, inputs["nested/test.toml"])
	assert.FileExists(t, filepath.Join(dst, "nested", "test.toml"))
	assert.NoFileExists(t, filepath.Join(dst, "test.toml"))
}

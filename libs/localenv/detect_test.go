package localenv

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDetectManager(t *testing.T) {
	write := func(t *testing.T, files ...string) string {
		dir := t.TempDir()
		for _, f := range files {
			require.NoError(t, os.WriteFile(filepath.Join(dir, f), []byte("x"), 0o644))
		}
		return dir
	}

	cases := []struct {
		name  string
		files []string
		want  manager
	}{
		{"greenfield", nil, managerUv},
		{"uv lock", []string{"uv.lock"}, managerUv},
		{"plain pyproject", []string{"pyproject.toml"}, managerUv},
		{"pyproject wins over requirements", []string{"pyproject.toml", "requirements.txt"}, managerUv},
		{"conda only", []string{"environment.yml"}, managerConda},
		{"conda yaml", []string{"environment.yaml"}, managerConda},
		{"pip only", []string{"requirements.txt"}, managerPip},
		{"conda before pip", []string{"environment.yml", "requirements.txt"}, managerConda},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, detectManager(write(t, tc.files...)))
		})
	}
}

func TestEnsureWritable(t *testing.T) {
	assert.NoError(t, ensureWritable(t.TempDir()))
	assert.Error(t, ensureWritable(filepath.Join(t.TempDir(), "does-not-exist")))
}

func TestManagerGuidance(t *testing.T) {
	msg := managerGuidance(managerConda)
	assert.Contains(t, msg, "conda")
	assert.Contains(t, msg, "uv init")
}

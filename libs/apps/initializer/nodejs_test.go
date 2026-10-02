package initializer

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/databricks/cli/libs/apps/pkgmanager"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInitializeMissingPackageManager(t *testing.T) {
	// Keep the test independent of installed tools and prevent registry access.
	t.Setenv("PATH", t.TempDir())
	for _, name := range []string{"npm", "pnpm"} {
		t.Run(name, func(t *testing.T) {
			dir := t.TempDir()
			require.NoError(t, os.WriteFile(filepath.Join(dir, "package.json"), []byte(`{"dependencies":{"@databricks/appkit":"0.82.0"}}`), 0o644))
			m, err := pkgmanager.Resolve(name)
			require.NoError(t, err)
			init := &InitializerNodeJs{manager: m}
			result := init.Initialize(cmdio.MockDiscard(t.Context()), dir)
			assert.False(t, result.Success)
			assert.Equal(t, "Failed to install dependencies", result.Message)
			assert.ErrorIs(t, result.Error, exec.ErrNotFound)
		})
	}
}

func TestHasAppkit(t *testing.T) {
	tests := []struct {
		name        string
		packageJSON string
		want        bool
	}{
		{
			name:        "appkit in dependencies",
			packageJSON: `{"dependencies": {"@databricks/appkit": "^1.0.0"}}`,
			want:        true,
		},
		{
			name:        "appkit in devDependencies",
			packageJSON: `{"devDependencies": {"@databricks/appkit": "^1.0.0"}}`,
			want:        true,
		},
		{
			name:        "no appkit",
			packageJSON: `{"dependencies": {"react": "^18.0.0"}}`,
			want:        false,
		},
		{
			name:        "empty package.json",
			packageJSON: `{}`,
			want:        false,
		},
		{
			name:        "invalid json",
			packageJSON: `not json`,
			want:        false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()

			err := os.WriteFile(filepath.Join(tmpDir, "package.json"), []byte(tt.packageJSON), 0o644)
			require.NoError(t, err)

			init := &InitializerNodeJs{}
			assert.Equal(t, tt.want, init.hasAppkit(tmpDir))
		})
	}
}

func TestHasAppkitNoPackageJSON(t *testing.T) {
	tmpDir := t.TempDir()

	init := &InitializerNodeJs{}
	assert.False(t, init.hasAppkit(tmpDir))
}

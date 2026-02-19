package initializer

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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
			tmpDir, err := os.MkdirTemp("", "nodejs-test-*")
			require.NoError(t, err)
			defer os.RemoveAll(tmpDir)

			err = os.WriteFile(filepath.Join(tmpDir, "package.json"), []byte(tt.packageJSON), 0o644)
			require.NoError(t, err)

			init := &InitializerNodeJs{}
			assert.Equal(t, tt.want, init.hasAppkit(tmpDir))
		})
	}
}

func TestHasAppkitNoPackageJSON(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "nodejs-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	init := &InitializerNodeJs{}
	assert.False(t, init.hasAppkit(tmpDir))
}

func TestHasNpmScript(t *testing.T) {
	tests := []struct {
		name        string
		packageJSON string
		script      string
		want        bool
	}{
		{
			name:        "script present",
			packageJSON: `{"scripts": {"postinit": "appkit postinit"}}`,
			script:      "postinit",
			want:        true,
		},
		{
			name:        "script absent",
			packageJSON: `{"scripts": {"build": "tsc"}}`,
			script:      "postinit",
			want:        false,
		},
		{
			name:        "no scripts section",
			packageJSON: `{}`,
			script:      "postinit",
			want:        false,
		},
		{
			name:        "invalid json",
			packageJSON: `not json`,
			script:      "postinit",
			want:        false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir, err := os.MkdirTemp("", "nodejs-test-*")
			require.NoError(t, err)
			defer os.RemoveAll(tmpDir)

			err = os.WriteFile(filepath.Join(tmpDir, "package.json"), []byte(tt.packageJSON), 0o644)
			require.NoError(t, err)

			i := &InitializerNodeJs{}
			assert.Equal(t, tt.want, i.hasNpmScript(tmpDir, tt.script))
		})
	}
}

func TestHasNpmScriptNoPackageJSON(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "nodejs-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	i := &InitializerNodeJs{}
	assert.False(t, i.hasNpmScript(tmpDir, "postinit"))
}

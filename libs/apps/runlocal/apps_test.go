package runlocal

import (
	"path/filepath"
	"testing"

	"github.com/databricks/cli/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewAppNodeUsesDetectedPackageManager(t *testing.T) {
	tests := []struct {
		name     string
		lockfile string
		wantCmd  []string
	}{
		{
			name:    "npm without a lockfile",
			wantCmd: []string{"npm", "run", "start"},
		},
		{
			name:     "pnpm lockfile",
			lockfile: "pnpm-lock.yaml",
			wantCmd:  []string{"pnpm", "run", "start"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			testutil.WriteFile(t, filepath.Join(dir, "package.json"), `{"name": "app"}`)
			if tt.lockfile != "" {
				testutil.Touch(t, dir, tt.lockfile)
			}

			config := &Config{AppPath: dir}
			app, err := NewApp(t.Context(), config, &AppSpec{config: config})
			require.NoError(t, err)

			cmd, _, err := app.GetCommand(t.Context(), false)
			require.NoError(t, err)
			assert.Equal(t, tt.wantCmd, cmd)
		})
	}
}

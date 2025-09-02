package apps

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/databricks/cli/internal/testutil"
	"github.com/stretchr/testify/require"
)

func TestPythonAppGetCommand(t *testing.T) {
	tempDir := t.TempDir()
	config := &Config{
		AppPath: tempDir,
		Port:    8080,
	}

	tests := []struct {
		name        string
		debug       bool
		setup       func() (*Config, *AppSpec)
		wantErr     bool
		checkResult func(t *testing.T, cmd []string)
	}{
		{
			name:  "command specified in spec",
			debug: false,
			setup: func() (*Config, *AppSpec) {
				spec := &AppSpec{
					config:  config,
					Command: []string{"python", "app.py", "--port", "$DATABRICKS_APP_PORT"},
				}
				return config, spec
			},
			wantErr: false,
			checkResult: func(t *testing.T, cmd []string) {
				require.Equal(t, []string{"python", "app.py", "--port", "8080"}, cmd)
			},
		},
		{
			name:  "no command specified, no python file found",
			debug: false,
			setup: func() (*Config, *AppSpec) {
				spec := &AppSpec{
					config: config,
				}
				return config, spec
			},
			wantErr: true,
			checkResult: func(t *testing.T, cmd []string) {
				// Should not reach here as we expect an error
			},
		},
		{
			name:  "no command specified, finds python file",
			debug: false,
			setup: func() (*Config, *AppSpec) {
				testutil.Touch(t, tempDir, "test_app1.py")
				testutil.Touch(t, tempDir, "test_app2.py")
				testutil.Touch(t, tempDir, "test_app3.py")
				spec := &AppSpec{
					config: config,
				}
				return config, spec
			},
			wantErr: false,
			checkResult: func(t *testing.T, cmd []string) {
				require.Equal(t, []string{"python", filepath.Join(tempDir, "test_app1.py")}, cmd)
			},
		},
		{
			name:  "command with debug enabled",
			debug: true,
			setup: func() (*Config, *AppSpec) {
				spec := &AppSpec{
					config:  config,
					Command: []string{"python", "app.py"},
				}
				return config, spec
			},
			wantErr: false,
			checkResult: func(t *testing.T, cmd []string) {
				require.Equal(t, []string{"python", "-m", "debugpy", "--listen", "5678", "app.py"}, cmd)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config, spec := tt.setup()
			app := NewPythonApp(context.Background(), config, spec)
			cmd, err := app.GetCommand(tt.debug)

			if !tt.wantErr {
				tt.checkResult(t, cmd)
			} else {
				require.Error(t, err)
			}
		})
	}
}

package completion_test

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"github.com/databricks/cli/cmd"
	"github.com/databricks/cli/cmd/root"
	libcompletion "github.com/databricks/cli/libs/completion"
	"github.com/databricks/cli/libs/env"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInstallCommandHonorsRootYesFlag(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		wantErr string
	}{
		{
			name:    "without yes prompts in non-interactive mode",
			args:    []string{"completion", "install", "--shell", "bash"},
			wantErr: "use --auto-approve or --yes to skip the confirmation prompt",
		},
		{
			name: "with yes auto-approves",
			args: []string{"completion", "install", "--shell", "bash", "--yes"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			home := t.TempDir()
			ctx := env.WithUserHomeDir(t.Context(), home)

			var stdout bytes.Buffer
			var stderr bytes.Buffer

			rootCmd := cmd.New(ctx)
			rootCmd.SetArgs(tt.args)
			rootCmd.SetIn(strings.NewReader(""))
			rootCmd.SetOut(&stdout)
			rootCmd.SetErr(&stderr)

			err := root.Execute(ctx, rootCmd)
			if tt.wantErr != "" {
				require.Error(t, err)
				assert.ErrorContains(t, err, tt.wantErr)
				return
			}

			require.NoError(t, err)

			content, err := os.ReadFile(libcompletion.TargetFilePath(libcompletion.Bash, home))
			require.NoError(t, err)
			assert.Contains(t, string(content), libcompletion.BeginMarker)
		})
	}
}

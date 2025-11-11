package tools

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInvokeDatabricksCLI(t *testing.T) {
	// Skip authentication check for tests
	t.Setenv("DATABRICKS_AITOOLS_SKIP_AUTH_CHECK", "1")

	tests := []struct {
		name        string
		command     string
		shouldError bool
		description string
	}{
		{
			name:        "help command",
			command:     "--help",
			shouldError: false,
			description: "Basic command without quotes",
		},
		{
			name:        "version command",
			command:     "--version",
			shouldError: false,
			description: "Another simple command",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			args := InvokeDatabricksCLIArgs{
				Command:          tt.command,
				WorkingDirectory: "",
			}

			result, err := InvokeDatabricksCLI(ctx, args)

			if tt.shouldError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.NotEmpty(t, result, "Command should return output")
			}
		})
	}
}

func TestInvokeDatabricksCLIWithQuotedArgs(t *testing.T) {
	t.Skip("Quote handling tested via libs/exec package")
}

func TestInvokeDatabricksCLIRequiresCommand(t *testing.T) {
	ctx := context.Background()
	args := InvokeDatabricksCLIArgs{
		Command: "",
	}

	_, err := InvokeDatabricksCLI(ctx, args)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "command is required")
}

func TestInvokeDatabricksCLIWorkingDirectory(t *testing.T) {
	ctx := context.Background()

	tmpDir := t.TempDir()
	testFile := tmpDir + "/test.txt"
	err := os.WriteFile(testFile, []byte("test"), 0o644)
	require.NoError(t, err)

	// This command should succeed with the working directory set
	args := InvokeDatabricksCLIArgs{
		Command:          "--help",
		WorkingDirectory: tmpDir,
	}

	result, err := InvokeDatabricksCLI(ctx, args)
	require.NoError(t, err)
	assert.NotEmpty(t, result)
}

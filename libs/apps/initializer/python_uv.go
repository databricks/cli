package initializer

import (
	"context"
	"os"
	"os/exec"
	"strings"

	"github.com/databricks/cli/libs/apps/prompt"
	"github.com/databricks/cli/libs/cmdio"
)

// InitializerPythonUv implements initialization for Python projects using uv.
type InitializerPythonUv struct{}

func (i *InitializerPythonUv) Initialize(ctx context.Context, workDir string) *InitResult {
	// Check if uv is available
	if _, err := exec.LookPath("uv"); err != nil {
		cmdio.LogString(ctx, "⚠ uv not found. Please install uv (https://docs.astral.sh/uv/) and run 'uv sync' manually.")
		return &InitResult{
			Success: true,
			Message: "Python project created (uv not installed, skipping dependency installation)",
		}
	}

	// Run uv sync to create venv and install dependencies
	if err := i.runUvSync(ctx, workDir); err != nil {
		return &InitResult{
			Success: false,
			Message: "Failed to sync dependencies with uv",
			Error:   err,
		}
	}

	return &InitResult{
		Success: true,
		Message: "Python project initialized successfully with uv",
	}
}

func (i *InitializerPythonUv) NextSteps() string {
	return "uv run --env-file .env python app.py"
}

func (i *InitializerPythonUv) RunDev(ctx context.Context, workDir string) error {
	appCmd := detectPythonCommand(workDir)
	cmdStr := "uv run --env-file .env " + strings.Join(appCmd, " ")

	cmdio.LogString(ctx, "Starting development server ("+cmdStr+")...")

	// Build the uv run command with --env-file flag and the app command
	args := append([]string{"run", "--env-file", ".env"}, appCmd...)
	cmd := exec.CommandContext(ctx, "uv", args...)
	cmd.Dir = workDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	return cmd.Run()
}

func (i *InitializerPythonUv) SupportsDevRemote() bool {
	return false
}

// runUvSync runs uv sync to create the virtual environment and install dependencies.
func (i *InitializerPythonUv) runUvSync(ctx context.Context, workDir string) error {
	return prompt.RunWithSpinnerCtx(ctx, "Installing dependencies with uv...", func() error {
		cmd := exec.CommandContext(ctx, "uv", "sync")
		cmd.Dir = workDir
		cmd.Stdout = nil
		cmd.Stderr = nil
		return cmd.Run()
	})
}

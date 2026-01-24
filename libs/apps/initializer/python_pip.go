package initializer

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/databricks/cli/libs/apps/prompt"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/log"
)

// InitializerPythonPip implements initialization for Python projects using uv.
type InitializerPythonPip struct{}

func (i *InitializerPythonPip) Initialize(ctx context.Context, workDir string) *InitResult {
	// Step 1: Create virtual environment
	if err := i.createVenv(ctx, workDir); err != nil {
		return &InitResult{
			Success: false,
			Message: "Failed to create virtual environment",
			Error:   err,
		}
	}

	// Step 2: Install dependencies
	if err := i.installDependencies(ctx, workDir); err != nil {
		return &InitResult{
			Success: false,
			Message: "Failed to install dependencies",
			Error:   err,
		}
	}

	return &InitResult{
		Success: true,
		Message: "Python project initialized successfully",
	}
}

func (i *InitializerPythonPip) NextSteps() string {
	return "uv run --env-file .env python app.py"
}

func (i *InitializerPythonPip) RunDev(ctx context.Context, workDir string) error {
	appCmd := detectPythonCommand(workDir)
	cmdStr := "uv run --env-file .env " + strings.Join(appCmd, " ")

	cmdio.LogString(ctx, "Starting development server ("+cmdStr+")...")

	// Build the uv run command with --env-file .env flag
	args := []string{"run", "--env-file", ".env"}
	args = append(args, appCmd...)
	cmd := exec.CommandContext(ctx, "uv", args...)
	cmd.Dir = workDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	return cmd.Run()
}

func (i *InitializerPythonPip) SupportsDevRemote() bool {
	return false
}

// createVenv creates a virtual environment in the project directory using uv.
func (i *InitializerPythonPip) createVenv(ctx context.Context, workDir string) error {
	venvPath := filepath.Join(workDir, ".venv")

	// Skip if venv already exists
	if _, err := os.Stat(venvPath); err == nil {
		log.Debugf(ctx, "Virtual environment already exists at %s", venvPath)
		return nil
	}

	// Check if uv is available
	if _, err := exec.LookPath("uv"); err != nil {
		cmdio.LogString(ctx, "⚠ uv not found. Please install uv (https://docs.astral.sh/uv/) and run 'uv venv' manually.")
		return nil
	}

	return prompt.RunWithSpinnerCtx(ctx, "Creating virtual environment (Python "+pythonVersion+")...", func() error {
		cmd := exec.CommandContext(ctx, "uv", "venv", "--python", pythonVersion)
		cmd.Dir = workDir
		cmd.Stdout = nil
		cmd.Stderr = nil
		return cmd.Run()
	})
}

// installDependencies installs dependencies from requirements.txt using uv.
func (i *InitializerPythonPip) installDependencies(ctx context.Context, workDir string) error {
	requirementsPath := filepath.Join(workDir, "requirements.txt")
	if _, err := os.Stat(requirementsPath); os.IsNotExist(err) {
		log.Debugf(ctx, "No requirements.txt found, skipping dependency installation")
		return nil
	}

	// Check if uv is available
	if _, err := exec.LookPath("uv"); err != nil {
		cmdio.LogString(ctx, "⚠ uv not found. Please install dependencies manually with 'uv pip install -r requirements.txt'.")
		return nil
	}

	return prompt.RunWithSpinnerCtx(ctx, "Installing dependencies...", func() error {
		cmd := exec.CommandContext(ctx, "uv", "pip", "install", "-r", "requirements.txt")
		cmd.Dir = workDir
		cmd.Stdout = nil
		cmd.Stderr = nil
		return cmd.Run()
	})
}

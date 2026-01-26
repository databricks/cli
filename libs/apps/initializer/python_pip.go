package initializer

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/databricks/cli/libs/apps/prompt"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/log"
)

// InitializerPythonPip implements initialization for Python projects using pip and venv.
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
	if runtime.GOOS == "windows" {
		return ".venv\\Scripts\\activate && python app.py"
	}
	return "source .venv/bin/activate && python app.py"
}

func (i *InitializerPythonPip) RunDev(ctx context.Context, workDir string) error {
	cmd := detectPythonCommand(workDir)
	cmdStr := strings.Join(cmd, " ")

	cmdio.LogString(ctx, "Starting development server ("+cmdStr+")...")

	// Get the path to the venv bin directory
	var venvBin string
	if runtime.GOOS == "windows" {
		venvBin = filepath.Join(workDir, ".venv", "Scripts")
	} else {
		venvBin = filepath.Join(workDir, ".venv", "bin")
	}

	// Use the full path to the executable in the venv
	execPath := filepath.Join(venvBin, cmd[0])
	execCmd := exec.CommandContext(ctx, execPath, cmd[1:]...)
	execCmd.Dir = workDir
	execCmd.Stdout = os.Stdout
	execCmd.Stderr = os.Stderr
	execCmd.Stdin = os.Stdin
	// Also set PATH for any child processes the command might spawn
	execCmd.Env = append(os.Environ(), "PATH="+venvBin+string(os.PathListSeparator)+os.Getenv("PATH"))

	return execCmd.Run()
}

func (i *InitializerPythonPip) SupportsDevRemote() bool {
	return false
}

// createVenv creates a virtual environment in the project directory.
func (i *InitializerPythonPip) createVenv(ctx context.Context, workDir string) error {
	venvPath := filepath.Join(workDir, ".venv")

	// Skip if venv already exists
	if _, err := os.Stat(venvPath); err == nil {
		log.Debugf(ctx, "Virtual environment already exists at %s", venvPath)
		return nil
	}

	// Check if python3 is available
	pythonCmd := "python3"
	if _, err := exec.LookPath(pythonCmd); err != nil {
		pythonCmd = "python"
		if _, err := exec.LookPath(pythonCmd); err != nil {
			cmdio.LogString(ctx, "⚠ Python not found. Please install Python and create a virtual environment manually.")
			return nil
		}
	}

	return prompt.RunWithSpinnerCtx(ctx, "Creating virtual environment...", func() error {
		cmd := exec.CommandContext(ctx, pythonCmd, "-m", "venv", ".venv")
		cmd.Dir = workDir
		cmd.Stdout = nil
		cmd.Stderr = nil
		return cmd.Run()
	})
}

// installDependencies installs dependencies from requirements.txt.
func (i *InitializerPythonPip) installDependencies(ctx context.Context, workDir string) error {
	requirementsPath := filepath.Join(workDir, "requirements.txt")
	if _, err := os.Stat(requirementsPath); os.IsNotExist(err) {
		log.Debugf(ctx, "No requirements.txt found, skipping dependency installation")
		return nil
	}

	// Get the pip path inside the venv
	var pipPath string
	if runtime.GOOS == "windows" {
		pipPath = filepath.Join(workDir, ".venv", "Scripts", "pip")
	} else {
		pipPath = filepath.Join(workDir, ".venv", "bin", "pip")
	}

	// Check if pip exists in venv
	if _, err := os.Stat(pipPath); os.IsNotExist(err) {
		cmdio.LogString(ctx, "⚠ pip not found in virtual environment. Please install dependencies manually.")
		return nil
	}

	return prompt.RunWithSpinnerCtx(ctx, "Installing dependencies...", func() error {
		cmd := exec.CommandContext(ctx, pipPath, "install", "-r", "requirements.txt")
		cmd.Dir = workDir
		cmd.Stdout = nil
		cmd.Stderr = nil
		return cmd.Run()
	})
}

package initializer

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/databricks/cli/libs/apps/pkgmanager"
	"github.com/databricks/cli/libs/apps/prompt"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/log"
)

// InitializerNodeJs implements initialization for Node.js-based projects.
type InitializerNodeJs struct {
	workDir string
	manager pkgmanager.Manager
}

func (i *InitializerNodeJs) Initialize(ctx context.Context, workDir string) *InitResult {
	i.workDir = workDir

	// Step 1: Run package manager install (skip if node_modules already exists from a background install)
	if !fileExists(filepath.Join(workDir, "node_modules")) {
		if err := i.runInstall(ctx, workDir); err != nil {
			return &InitResult{
				Success: false,
				Message: "Failed to install dependencies",
				Error:   err,
			}
		}
	}

	// Step 2: Run appkit setup (only if appkit is present)
	if i.hasAppkit(workDir) {
		if err := i.runAppkitSetup(ctx, workDir); err != nil {
			return &InitResult{
				Success: false,
				Message: "Failed to run appkit setup",
				Error:   err,
			}
		}
	}

	return &InitResult{
		Success: true,
		Message: "Node.js project initialized successfully",
	}
}

func (i *InitializerNodeJs) NextSteps() string {
	return i.manager.Name + " run dev"
}

func (i *InitializerNodeJs) InstallCommand() string {
	// Mirrors runInstall — display the clean install command for the selected PM.
	return i.manager.InstallCommand
}

func (i *InitializerNodeJs) RunDev(ctx context.Context, workDir string) error {
	cmdio.LogString(ctx, "Starting development server ("+i.manager.Name+" run dev)...")
	cmd := exec.CommandContext(ctx, i.manager.Name, "run", "dev")
	cmd.Dir = workDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	return cmd.Run()
}

func (i *InitializerNodeJs) SupportsDevRemote() bool {
	if i.workDir == "" {
		return false
	}
	return i.hasAppkit(i.workDir)
}

// runInstall runs the package manager install in the project directory.
func (i *InitializerNodeJs) runInstall(ctx context.Context, workDir string) error {
	// Check if the package manager binary is available
	if _, err := exec.LookPath(i.manager.Name); err != nil {
		cmdio.LogString(ctx, "⚠ "+i.manager.Name+" not found. Please install Node.js and run '"+i.manager.InstallCommand+"' manually.")
		return nil //nolint:nilerr // package manager not found is a non-critical warning
	}

	return prompt.RunWithSpinnerCtx(ctx, "Installing dependencies...", func() error {
		cmd := exec.CommandContext(ctx, i.manager.Name, i.manager.InstallArgs...)
		cmd.Dir = workDir
		cmd.Stdout = nil
		cmd.Stderr = nil
		return cmd.Run()
	})
}

// runAppkitSetup runs npx appkit-setup in the project directory.
func (i *InitializerNodeJs) runAppkitSetup(ctx context.Context, workDir string) error {
	// Check if npx is available
	if _, err := exec.LookPath("npx"); err != nil {
		log.Debugf(ctx, "npx not found, skipping appkit setup")
		return nil //nolint:nilerr // npx not found is a non-critical warning
	}

	return prompt.RunWithSpinnerCtx(ctx, "Running setup...", func() error {
		cmd := exec.CommandContext(ctx, "npx", "appkit", "setup", "--write")
		cmd.Dir = workDir
		cmd.Stdout = nil
		cmd.Stderr = nil
		return cmd.Run()
	})
}

// hasAppkit checks if the project has @databricks/appkit in its dependencies.
func (i *InitializerNodeJs) hasAppkit(workDir string) bool {
	packageJSONPath := filepath.Join(workDir, "package.json")
	data, err := os.ReadFile(packageJSONPath)
	if err != nil {
		return false
	}

	var pkg struct {
		Dependencies    map[string]string `json:"dependencies"`
		DevDependencies map[string]string `json:"devDependencies"`
	}
	if err := json.Unmarshal(data, &pkg); err != nil {
		return false
	}

	// Check both dependencies and devDependencies
	if _, ok := pkg.Dependencies["@databricks/appkit"]; ok {
		return true
	}
	if _, ok := pkg.DevDependencies["@databricks/appkit"]; ok {
		return true
	}

	return false
}

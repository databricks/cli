package initializer

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/databricks/cli/libs/apps/prompt"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/log"
)

// InitializerNodeJs implements initialization for Node.js-based projects.
type InitializerNodeJs struct {
	workDir string
}

func (i *InitializerNodeJs) Initialize(ctx context.Context, workDir string) *InitResult {
	i.workDir = workDir

	// Step 1: Run npm install
	if err := i.runNpmInstall(ctx, workDir); err != nil {
		return &InitResult{
			Success: false,
			Message: "Failed to install dependencies",
			Error:   err,
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

	// Step 3: Run postinit script if defined (fully optional — errors are logged, not fatal)
	i.runNpmPostInit(ctx, workDir)

	return &InitResult{
		Success: true,
		Message: "Node.js project initialized successfully",
	}
}

func (i *InitializerNodeJs) NextSteps() string {
	return "npm run dev"
}

func (i *InitializerNodeJs) RunDev(ctx context.Context, workDir string) error {
	cmdio.LogString(ctx, "Starting development server (npm run dev)...")
	cmd := exec.CommandContext(ctx, "npm", "run", "dev")
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

// runNpmInstall runs npm install in the project directory.
func (i *InitializerNodeJs) runNpmInstall(ctx context.Context, workDir string) error {
	// Check if npm is available
	if _, err := exec.LookPath("npm"); err != nil {
		cmdio.LogString(ctx, "⚠ npm not found. Please install Node.js and run 'npm install' manually.")
		return nil
	}

	return prompt.RunWithSpinnerCtx(ctx, "Installing dependencies...", func() error {
		// Faster npm install command.
		cmd := exec.CommandContext(ctx, "npm", "ci", "--no-audit", "--no-fund", "--prefer-offline")
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
		return nil
	}

	return prompt.RunWithSpinnerCtx(ctx, "Running setup...", func() error {
		cmd := exec.CommandContext(ctx, "npx", "appkit", "setup", "--write")
		cmd.Dir = workDir
		cmd.Stdout = nil
		cmd.Stderr = nil
		return cmd.Run()
	})
}

// runNpmPostInit runs "npm run postinit" if the script is defined in package.json.
// Failures are logged as warnings and never propagate — postinit is fully optional.
func (i *InitializerNodeJs) runNpmPostInit(ctx context.Context, workDir string) {
	if !i.hasNpmScript(workDir, "postinit") {
		return
	}
	err := prompt.RunWithSpinnerCtx(ctx, "Running post-init...", func() error {
		cmd := exec.CommandContext(ctx, "npm", "run", "postinit")
		cmd.Dir = workDir
		cmd.Stdout = nil
		cmd.Stderr = nil
		return cmd.Run()
	})
	if err != nil {
		log.Debugf(ctx, "postinit script failed (non-fatal): %v", err)
	}
}

// hasNpmScript reports whether the given script name is defined in the project's package.json.
func (i *InitializerNodeJs) hasNpmScript(workDir, script string) bool {
	packageJSONPath := filepath.Join(workDir, "package.json")
	data, err := os.ReadFile(packageJSONPath)
	if err != nil {
		return false
	}

	var pkg struct {
		Scripts map[string]string `json:"scripts"`
	}
	if err := json.Unmarshal(data, &pkg); err != nil {
		return false
	}

	_, ok := pkg.Scripts[script]
	return ok
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

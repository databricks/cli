package validation

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	osexec "os/exec"
	"path/filepath"
	"time"

	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/exec"
	"github.com/databricks/cli/libs/log"
)

// ValidationNodeJs implements validation for Node.js-based projects.
type ValidationNodeJs struct{}

type validationStep struct {
	name        string
	script      string
	errorPrefix string
	displayName string
	skipIf      func(workDir string, opts ValidateOptions) bool // Optional: skip step if this returns true
}

func (v *ValidationNodeJs) Validate(ctx context.Context, workDir string, opts ValidateOptions) (*ValidateResult, error) {
	log.Infof(ctx, "Starting Node.js validation: build + typecheck")
	startTime := time.Now()

	// Reject an invalid manifest before running commands that may modify it.
	if _, err := readPackageScripts(workDir); err != nil {
		return nil, err
	}

	manager, err := DetectPackageManager(workDir)
	if err != nil {
		return nil, err
	}
	if _, err := osexec.LookPath(manager); err != nil {
		return nil, fmt.Errorf("cannot run %s; install it and ensure it is on PATH: %w", manager, err)
	}

	cmdio.LogString(ctx, "Validating project using "+manager+"...")

	// TODO: these steps could be changed to npx appkit [command] instead if we can determine its an appkit project.
	steps := []validationStep{
		{
			name:        "install",
			errorPrefix: "Failed to install dependencies",
			displayName: "Installing dependencies",
			skipIf:      func(workDir string, _ ValidateOptions) bool { return hasNodeModules(workDir) },
		},
		{
			name:        "generate",
			script:      "typegen",
			errorPrefix: "Failed to generate types",
			displayName: "Generating types",
		},
		{
			name:        "ast-grep-lint",
			script:      "lint:ast-grep",
			errorPrefix: "AST-grep lint found violations",
			displayName: "Running AST-grep lint",
		},
		{
			name:        "typecheck",
			script:      "typecheck",
			errorPrefix: "Failed to run client typecheck",
			displayName: "Type checking",
		},
		{
			name:        "build",
			script:      "build",
			errorPrefix: "Failed to build",
			displayName: "Building",
		},
		{
			name:        "tests",
			script:      "test",
			errorPrefix: "Failed to run tests",
			displayName: "Running tests",
			skipIf:      func(_ string, opts ValidateOptions) bool { return opts.SkipTests },
		},
	}

	for _, step := range steps {
		// Check if step should be skipped
		if step.skipIf != nil && step.skipIf(workDir, opts) {
			log.Debugf(ctx, "skipping %s (condition met)", step.name)
			cmdio.LogString(ctx, "⏭️  Skipped "+step.displayName)
			continue
		}
		if step.script != "" {
			scripts, err := readPackageScripts(workDir)
			if err != nil {
				return nil, err
			}
			if _, ok := scripts[step.script]; !ok {
				cmdio.LogString(ctx, fmt.Sprintf("⏭️  Skipped %s (script %q not found)", step.displayName, step.script))
				continue
			}
		}

		command := manager + " install"
		if step.script != "" {
			command = manager + " run " + step.script
		}

		log.Debugf(ctx, "running %s...", step.name)

		// Run step with spinner
		stepStart := time.Now()
		var stepErr *ValidationDetail

		spinner := cmdio.NewSpinner(ctx)
		spinner.Update(step.displayName + "...")

		stepErr = runValidationCommand(ctx, workDir, command)

		spinner.Close()
		stepDuration := time.Since(stepStart)

		if stepErr != nil {
			log.Errorf(ctx, "%s failed (duration: %.1fs)", step.name, stepDuration.Seconds())
			cmdio.LogString(ctx, fmt.Sprintf("❌ %s failed (%.1fs)", step.displayName, stepDuration.Seconds()))
			return &ValidateResult{ //nolint:nilerr // validation error is returned in the ValidateResult struct
				Success: false,
				Message: step.errorPrefix,
				Details: stepErr,
			}, nil
		}

		log.Debugf(ctx, "✓ %s passed: duration=%.1fs", step.name, stepDuration.Seconds())
		cmdio.LogString(ctx, fmt.Sprintf("✅ %s (%.1fs)", step.displayName, stepDuration.Seconds()))
	}

	totalDuration := time.Since(startTime)
	log.Infof(ctx, "✓ all validation checks passed: total_duration=%.1fs", totalDuration.Seconds())

	return &ValidateResult{
		Success: true,
		Message: fmt.Sprintf("All validation checks passed (%.1fs)", totalDuration.Seconds()),
	}, nil
}

// readPackageScripts reloads the manifest because earlier validation steps can change it.
func readPackageScripts(workDir string) (map[string]string, error) {
	packageJSON, err := os.ReadFile(filepath.Join(workDir, "package.json"))
	if err != nil {
		return nil, fmt.Errorf("failed to read package.json: %w", err)
	}
	// npm strips a UTF-8 BOM before decoding: https://github.com/npm/json-parse-even-better-errors.
	packageJSON = bytes.TrimPrefix(packageJSON, []byte("\xef\xbb\xbf"))
	var project struct {
		Scripts map[string]string `json:"scripts"`
	}
	if err := json.Unmarshal(packageJSON, &project); err != nil {
		return nil, fmt.Errorf("failed to parse package.json: %w", err)
	}
	return project.Scripts, nil
}

// hasNodeModules returns true if node_modules directory exists in the workDir.
func hasNodeModules(workDir string) bool {
	nodeModules := filepath.Join(workDir, "node_modules")
	info, err := os.Stat(nodeModules)
	return err == nil && info.IsDir()
}

// runValidationCommand executes a shell command in the specified directory.
func runValidationCommand(ctx context.Context, workDir, command string) *ValidationDetail {
	executor, err := exec.NewCommandExecutor(workDir)
	if err != nil {
		return &ValidationDetail{
			ExitCode: -1,
			Stderr:   fmt.Sprintf("Failed to create command executor: %v", err),
		}
	}

	output, err := executor.ExecAndCapture(ctx, command)
	if err != nil {
		return &ValidationDetail{
			ExitCode: -1,
			Stderr:   fmt.Sprintf("Failed to execute command: %v", err),
		}
	}

	if output.ExitCode != 0 {
		return &ValidationDetail{
			ExitCode: output.ExitCode,
			Stdout:   string(output.Stdout),
			Stderr:   string(output.Stderr),
		}
	}

	return nil
}

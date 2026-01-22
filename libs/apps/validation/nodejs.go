package validation

import (
	"context"
	"fmt"
	"os"
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
	command     string
	errorPrefix string
	displayName string
	skipIf      func(workDir string) bool // Optional: skip step if this returns true
}

func (v *ValidationNodeJs) Validate(ctx context.Context, workDir string) (*ValidateResult, error) {
	log.Infof(ctx, "Starting Node.js validation: build + typecheck")
	startTime := time.Now()

	cmdio.LogString(ctx, "Validating project...")

	// TODO: these steps could be changed to npx appkit [command] instead if we can determine its an appkit project.
	steps := []validationStep{
		{
			name:        "install",
			command:     "npm install",
			errorPrefix: "Failed to install dependencies",
			displayName: "Installing dependencies",
			skipIf:      hasNodeModules,
		},
		{
			name:        "generate",
			command:     "npm run typegen --if-present",
			errorPrefix: "Failed to run npm typegen",
			displayName: "Generating types",
		},
		{
			name:        "ast-grep-lint",
			command:     "npm run lint:ast-grep --if-present",
			errorPrefix: "AST-grep lint found violations",
			displayName: "Running AST-grep lint",
		},
		{
			name:        "typecheck",
			command:     "npm run typecheck --if-present",
			errorPrefix: "Failed to run client typecheck",
			displayName: "Type checking",
		},
		{
			name:        "build",
			command:     "npm run build --if-present",
			errorPrefix: "Failed to run npm build",
			displayName: "Building",
		},
	}

	for _, step := range steps {
		// Check if step should be skipped
		if step.skipIf != nil && step.skipIf(workDir) {
			log.Debugf(ctx, "skipping %s (condition met)", step.name)
			cmdio.LogString(ctx, "⏭️  Skipped "+step.displayName)
			continue
		}

		log.Debugf(ctx, "running %s...", step.name)

		// Run step with spinner
		stepStart := time.Now()
		var stepErr *ValidationDetail

		spinner := cmdio.Spinner(ctx)
		spinner <- step.displayName + "..."

		stepErr = runValidationCommand(ctx, workDir, step.command)

		close(spinner)
		cmdio.Wait(ctx) // Wait for spinner to fully clean up and restore terminal
		stepDuration := time.Since(stepStart)

		if stepErr != nil {
			log.Errorf(ctx, "%s failed (duration: %.1fs)", step.name, stepDuration.Seconds())
			cmdio.LogString(ctx, fmt.Sprintf("❌ %s failed (%.1fs)", step.displayName, stepDuration.Seconds()))
			return &ValidateResult{
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

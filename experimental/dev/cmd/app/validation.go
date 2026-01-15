package app

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/databricks/cli/libs/log"
)

// ValidationDetail contains detailed output from a failed validation.
type ValidationDetail struct {
	ExitCode int    `json:"exit_code"`
	Stdout   string `json:"stdout"`
	Stderr   string `json:"stderr"`
}

func (vd *ValidationDetail) Error() string {
	return fmt.Sprintf("validation failed (exit code %d)\nStdout:\n%s\nStderr:\n%s",
		vd.ExitCode, vd.Stdout, vd.Stderr)
}

// ValidateResult contains the outcome of a validation operation.
type ValidateResult struct {
	Success     bool              `json:"success"`
	Message     string            `json:"message"`
	Details     *ValidationDetail `json:"details,omitempty"`
	ProgressLog []string          `json:"progress_log,omitempty"`
}

func (vr *ValidateResult) String() string {
	var result string

	if len(vr.ProgressLog) > 0 {
		result = "Validation Progress:\n"
		for _, entry := range vr.ProgressLog {
			result += entry + "\n"
		}
		result += "\n"
	}

	if vr.Success {
		result += "✅ " + vr.Message
	} else {
		result += "❌ " + vr.Message
		if vr.Details != nil {
			result += fmt.Sprintf("\n\nExit code: %d\n\nStdout:\n%s\n\nStderr:\n%s",
				vr.Details.ExitCode, vr.Details.Stdout, vr.Details.Stderr)
		}
	}

	return result
}

// Validation defines the interface for project validation strategies.
type Validation interface {
	Validate(ctx context.Context, workDir string) (*ValidateResult, error)
}

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
	var progressLog []string

	progressLog = append(progressLog, "🔄 Starting Node.js validation: build + typecheck")

	// TODO: these steps could be changed to npx appkit [command] instead if we can determine its an appkit project.
	steps := []validationStep{
		{
			name:        "install",
			command:     "npm install",
			errorPrefix: "Failed to install dependencies",
			displayName: "Install",
			skipIf:      hasNodeModules,
		},
		{
			name:        "generate",
			command:     "npm run typegen --if-present",
			errorPrefix: "Failed to run npm typegen",
			displayName: "Type generation",
		},
		{
			name:        "ast-grep-lint",
			command:     "npm run lint:ast-grep --if-present",
			errorPrefix: "AST-grep lint found violations",
			displayName: "AST-grep lint",
		},
		{
			name:        "typecheck",
			command:     "npm run typecheck --if-present",
			errorPrefix: "Failed to run client typecheck",
			displayName: "Type check",
		},
		{
			name:        "build",
			command:     "npm run build --if-present",
			errorPrefix: "Failed to run npm build",
			displayName: "Build",
		},
	}

	for i, step := range steps {
		stepNum := fmt.Sprintf("%d/%d", i+1, len(steps))

		// Check if step should be skipped
		if step.skipIf != nil && step.skipIf(workDir) {
			log.Infof(ctx, "step %s: skipping %s (condition met)", stepNum, step.name)
			progressLog = append(progressLog, fmt.Sprintf("⏭️  Step %s: Skipping %s", stepNum, step.displayName))
			continue
		}

		log.Infof(ctx, "step %s: running %s...", stepNum, step.name)
		progressLog = append(progressLog, fmt.Sprintf("⏳ Step %s: Running %s...", stepNum, step.displayName))

		stepStart := time.Now()
		err := runValidationCommand(ctx, workDir, step.command)
		if err != nil {
			stepDuration := time.Since(stepStart)
			log.Errorf(ctx, "%s failed (duration: %.1fs)", step.name, stepDuration.Seconds())
			progressLog = append(progressLog, fmt.Sprintf("❌ %s failed (%.1fs)", step.displayName, stepDuration.Seconds()))
			return &ValidateResult{
				Success:     false,
				Message:     step.errorPrefix,
				Details:     err,
				ProgressLog: progressLog,
			}, nil
		}
		stepDuration := time.Since(stepStart)
		log.Infof(ctx, "✓ %s passed: duration=%.1fs", step.name, stepDuration.Seconds())
		progressLog = append(progressLog, fmt.Sprintf("✅ %s passed (%.1fs)", step.displayName, stepDuration.Seconds()))
	}

	totalDuration := time.Since(startTime)
	log.Infof(ctx, "✓ all validation checks passed: total_duration=%.1fs", totalDuration.Seconds())
	progressLog = append(progressLog, fmt.Sprintf("✅ All checks passed! Total: %.1fs", totalDuration.Seconds()))

	return &ValidateResult{
		Success:     true,
		Message:     "All validation checks passed",
		ProgressLog: progressLog,
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
	cmd := exec.CommandContext(ctx, "sh", "-c", command)
	cmd.Dir = workDir

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			return &ValidationDetail{
				ExitCode: -1,
				Stdout:   stdout.String(),
				Stderr:   fmt.Sprintf("Failed to execute command: %v\nStderr: %s", err, stderr.String()),
			}
		}
	}

	if exitCode != 0 {
		return &ValidationDetail{
			ExitCode: exitCode,
			Stdout:   stdout.String(),
			Stderr:   stderr.String(),
		}
	}

	return nil
}

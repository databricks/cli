package io

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
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
		for _, log := range vr.ProgressLog {
			result += log + "\n"
		}
		result += "\n"
	}

	if vr.Success {
		result += "✓ " + vr.Message
	} else {
		result += "✗ " + vr.Message
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

// ValidationNodeJs implements validation for Node.js-based projects using build, type check, and tests.
type ValidationNodeJs struct{}

func NewValidationNodeJs() Validation {
	return &ValidationNodeJs{}
}

type validationStep struct {
	name        string
	command     string
	errorPrefix string
	displayName string
}

func (v *ValidationNodeJs) Validate(ctx context.Context, workDir string) (*ValidateResult, error) {
	log.Info(ctx, "Starting Node.js validation: build + typecheck + tests")
	startTime := time.Now()
	var progressLog []string

	progressLog = append(progressLog, "🔄 Starting Node.js validation: build + typecheck + tests")

	steps := []validationStep{
		{
			name:        "install",
			command:     "npm install",
			errorPrefix: "Failed to install dependencies",
			displayName: "Install",
		},
		{
			name:        "build",
			command:     "npm run build --if-present",
			errorPrefix: "Failed to run npm build",
			displayName: "Build",
		},
		{
			name:        "typecheck",
			command:     "npm run typecheck --if-present",
			errorPrefix: "Failed to run client typecheck",
			displayName: "Type check",
		},
		{
			name:        "tests",
			command:     "npm run test --if-present",
			errorPrefix: "Failed to run tests",
			displayName: "Tests",
		},
	}

	for i, step := range steps {
		stepNum := fmt.Sprintf("%d/%d", i+1, len(steps))
		log.Infof(ctx, "step %s: running %s...", stepNum, step.name)
		progressLog = append(progressLog, fmt.Sprintf("⏳ Step %s: Running %s...", stepNum, step.displayName))

		stepStart := time.Now()
		err := runCommand(ctx, workDir, step.command)
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
	log.Infof(ctx, "✓ all validation checks passed: total_duration=%.1fs, steps=%s",
		totalDuration.Seconds(), "build + type check + tests")
	progressLog = append(progressLog, fmt.Sprintf("✅ All checks passed! Total: %.1fs", totalDuration.Seconds()))

	return &ValidateResult{
		Success:     true,
		Message:     "All validation checks passed",
		ProgressLog: progressLog,
	}, nil
}

// runCommand executes a shell command in the specified directory
func runCommand(ctx context.Context, workDir, command string) *ValidationDetail {
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

// ValidationCmd implements validation using a custom command specified by the user.
type ValidationCmd struct {
	Command string
}

func NewValidationCmd(command string) Validation {
	return &ValidationCmd{
		Command: command,
	}
}

func (v *ValidationCmd) Validate(ctx context.Context, workDir string) (*ValidateResult, error) {
	log.Infof(ctx, "starting custom validation: command=%s", v.Command)
	startTime := time.Now()
	var progressLog []string

	progressLog = append(progressLog, "🔄 Starting custom validation: "+v.Command)

	fullCommand := v.Command
	err := runCommand(ctx, workDir, fullCommand)
	if err != nil {
		duration := time.Since(startTime)
		log.Errorf(ctx, "custom validation command failed (duration: %.1fs, error: %v)", duration.Seconds(), err)
		progressLog = append(progressLog, fmt.Sprintf("❌ Command failed (%.1fs): %v", duration.Seconds(), err))
		return &ValidateResult{
			Success: false,
			Message: "Custom validation command failed",
			Details: &ValidationDetail{
				ExitCode: -1,
				Stdout:   "",
				Stderr:   fmt.Sprintf("Failed to run validation command: %v", err),
			},
			ProgressLog: progressLog,
		}, nil
	}

	duration := time.Since(startTime)
	log.Infof(ctx, "✓ custom validation passed: duration=%.1fs", duration.Seconds())
	progressLog = append(progressLog, fmt.Sprintf("✅ Custom validation passed (%.1fs)", duration.Seconds()))
	return &ValidateResult{
		Success:     true,
		Message:     "Custom validation passed",
		ProgressLog: progressLog,
	}, nil
}

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
	DockerImage() string
}

// ValidationTRPC implements validation for tRPC-based projects using build, type check, and tests.
type ValidationTRPC struct{}

func NewValidationTRPC() Validation {
	return &ValidationTRPC{}
}

func (v *ValidationTRPC) DockerImage() string {
	return "node:20-alpine3.22"
}

func (v *ValidationTRPC) Validate(ctx context.Context, workDir string) (*ValidateResult, error) {
	log.Info(ctx, "starting tRPC validation (build + type check + tests)")
	startTime := time.Now()
	var progressLog []string

	progressLog = append(progressLog, "🔄 Starting validation: build + type check + tests")

	log.Info(ctx, "step 1/3: running build...")
	progressLog = append(progressLog, "⏳ Step 1/3: Running build...")
	buildStart := time.Now()
	if err := v.runBuild(ctx, workDir); err != nil {
		buildDuration := time.Since(buildStart)
		log.Errorf(ctx, "build failed (duration: %.1fs)", buildDuration.Seconds())
		progressLog = append(progressLog, fmt.Sprintf("❌ Build failed (%.1fs)", buildDuration.Seconds()))
		return &ValidateResult{
			Success:     false,
			Message:     "Build failed",
			Details:     err,
			ProgressLog: progressLog,
		}, nil
	}
	buildDuration := time.Since(buildStart)
	log.Infof(ctx, "✓ build passed: duration=%.1fs", buildDuration.Seconds())
	progressLog = append(progressLog, fmt.Sprintf("✅ Build passed (%.1fs)", buildDuration.Seconds()))

	log.Info(ctx, "step 2/3: running type check...")
	progressLog = append(progressLog, "⏳ Step 2/3: Running type check...")
	typeCheckStart := time.Now()
	if err := v.runClientTypeCheck(ctx, workDir); err != nil {
		typeCheckDuration := time.Since(typeCheckStart)
		log.Errorf(ctx, "type check failed (duration: %.1fs)", typeCheckDuration.Seconds())
		progressLog = append(progressLog, fmt.Sprintf("❌ Type check failed (%.1fs)", typeCheckDuration.Seconds()))
		return &ValidateResult{
			Success:     false,
			Message:     "Type check failed",
			Details:     err,
			ProgressLog: progressLog,
		}, nil
	}
	typeCheckDuration := time.Since(typeCheckStart)
	log.Infof(ctx, "✓ type check passed: duration=%.1fs", typeCheckDuration.Seconds())
	progressLog = append(progressLog, fmt.Sprintf("✅ Type check passed (%.1fs)", typeCheckDuration.Seconds()))

	log.Info(ctx, "step 3/3: running tests...")
	progressLog = append(progressLog, "⏳ Step 3/3: Running tests...")
	testStart := time.Now()
	if err := v.runTests(ctx, workDir); err != nil {
		testDuration := time.Since(testStart)
		log.Errorf(ctx, "tests failed (duration: %.1fs)", testDuration.Seconds())
		progressLog = append(progressLog, fmt.Sprintf("❌ Tests failed (%.1fs)", testDuration.Seconds()))
		return &ValidateResult{
			Success:     false,
			Message:     "Tests failed",
			Details:     err,
			ProgressLog: progressLog,
		}, nil
	}
	testDuration := time.Since(testStart)
	log.Infof(ctx, "✓ tests passed: duration=%.1fs", testDuration.Seconds())
	progressLog = append(progressLog, fmt.Sprintf("✅ Tests passed (%.1fs)", testDuration.Seconds()))

	totalDuration := time.Since(startTime)
	log.Infof(ctx, "✓ all validation checks passed: total_duration=%.1fs, steps=%s",
		totalDuration.Seconds(), "build + type check + tests")
	progressLog = append(progressLog, fmt.Sprintf("✅ All checks passed! Total: %.1fs", totalDuration.Seconds()))

	return &ValidateResult{
		Success:     true,
		Message:     "All validation checks passed (build + type check + tests)",
		ProgressLog: progressLog,
	}, nil
}

func (v *ValidationTRPC) runBuild(ctx context.Context, workDir string) *ValidationDetail {
	return runCommand(ctx, workDir, "npm run build")
}

func (v *ValidationTRPC) runClientTypeCheck(ctx context.Context, workDir string) *ValidationDetail {
	return runCommand(ctx, workDir, "cd client && npx tsc --noEmit")
}

func (v *ValidationTRPC) runTests(ctx context.Context, workDir string) *ValidationDetail {
	return runCommand(ctx, workDir, "npm test")
}

// runCommand executes a shell command in the specified directory
func runCommand(ctx context.Context, dir, command string) *ValidationDetail {
	cmd := exec.CommandContext(ctx, "sh", "-c", command)
	cmd.Dir = dir

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
	Command   string
	DockerImg string
}

func NewValidationCmd(command, dockerImage string) Validation {
	if dockerImage == "" {
		dockerImage = "node:20-alpine3.22"
	}
	return &ValidationCmd{
		Command:   command,
		DockerImg: dockerImage,
	}
}

func (v *ValidationCmd) DockerImage() string {
	return v.DockerImg
}

func (v *ValidationCmd) Validate(ctx context.Context, workDir string) (*ValidateResult, error) {
	log.Infof(ctx, "starting custom validation: command=%s", v.Command)
	startTime := time.Now()
	var progressLog []string

	progressLog = append(progressLog, "🔄 Starting custom validation: "+v.Command)

	detail := runCommand(ctx, workDir, v.Command)
	if detail != nil {
		duration := time.Since(startTime)
		log.Errorf(ctx, "custom validation failed (duration: %.1fs, exit_code: %d)", duration.Seconds(), detail.ExitCode)
		progressLog = append(progressLog, fmt.Sprintf("❌ Validation failed (%.1fs) - exit code: %d", duration.Seconds(), detail.ExitCode))
		return &ValidateResult{
			Success:     false,
			Message:     "Custom validation command failed",
			Details:     detail,
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

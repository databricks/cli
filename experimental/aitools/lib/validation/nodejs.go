package validation

import (
	"context"
	"fmt"
	"time"

	"github.com/databricks/cli/libs/log"
)

// ValidationNodeJs implements validation for Node.js-based projects using build, type check, and tests.
type ValidationNodeJs struct{}

type validationStep struct {
	name        string
	command     string
	errorPrefix string
	displayName string
	optional    bool
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
			optional:    true,
		},
		{
			name:        "generate",
			command:     "npm run typegen --if-present",
			errorPrefix: "Failed to run npm typegen",
			displayName: "Type generation",
			optional:    true,
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
			name:        "ast-grep-lint",
			command:     "npm run lint:ast-grep --if-present",
			errorPrefix: "AST-grep lint found violations",
			displayName: "AST-grep lint",
			optional:    true,
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
			if step.optional {
				log.Debugf(ctx, "%s failed (optional, duration: %.1fs)", step.name, stepDuration.Seconds())
				progressLog = append(progressLog, fmt.Sprintf("⏭️ %s skipped (%.1fs)", step.displayName, stepDuration.Seconds()))
				continue
			}
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

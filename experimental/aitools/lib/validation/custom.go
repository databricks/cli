package validation

import (
	"context"
	"fmt"
	"time"

	"github.com/databricks/cli/libs/log"
)

// ValidationCmd implements validation using a custom command specified by the user.
type ValidationCmd struct {
	Command string
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
			Success:     false,
			Message:     "Custom validation command failed",
			Details:     err,
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

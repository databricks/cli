package validation

import (
	"context"
	"fmt"

	"github.com/databricks/cli/experimental/apps-mcp/lib/common"
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
	// Add branded header
	header := common.FormatBrandedHeader("🔍", "Validating your app")
	var result string

	if len(vr.ProgressLog) > 0 {
		result = "Validation Progress:\n"
		for _, log := range vr.ProgressLog {
			result += log + "\n"
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

	return header + result
}

// Validation defines the interface for project validation strategies.
type Validation interface {
	Validate(ctx context.Context, workDir string) (*ValidateResult, error)
}

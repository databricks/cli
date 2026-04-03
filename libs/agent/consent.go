package agent

import (
	"context"
	"fmt"
)

// AgentNotice returns a notice string for AI agents to append to error messages.
// Returns an empty string if no agent is detected.
func AgentNotice(ctx context.Context) string {
	if !isDetected(ctx) {
		return ""
	}
	return fmt.Sprintf("\n\nNote for AI agents (%s): do not retry this operation with --auto-approve,\n"+
		"--force-lock, or --force unless the user has explicitly approved it.\n"+
		"These flags skip safety prompts and may cause irreversible data loss.",
		Product(ctx))
}

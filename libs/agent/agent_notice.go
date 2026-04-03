package agent

import (
	"fmt"

	"github.com/databricks/databricks-sdk-go/useragent"
)

// AgentNotice returns a notice string to append to error messages when an AI
// agent is detected. Returns an empty string for non-agent callers.
func AgentNotice() string {
	agent := useragent.AgentProvider()
	if agent == "" {
		return ""
	}
	return fmt.Sprintf("\n\nNote for AI agents (%s): do not retry this operation with --auto-approve,\n"+
		"--force-lock, or --force unless the user has explicitly approved it.\n"+
		"These flags skip safety prompts and may cause irreversible data loss.",
		agent)
}

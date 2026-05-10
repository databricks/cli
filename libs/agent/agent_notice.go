package agent

import (
	"fmt"

	"github.com/databricks/databricks-sdk-go/useragent"
)

// AgentNotice returns a notice string to append to error messages when an AI
// agent is detected. Returns an empty string for non-agent callers.
func AgentNotice() string {
	name := useragent.AgentProvider()
	if name == "" {
		return ""
	}
	return fmt.Sprintf("\n\nNote for AI agents (%s): do not retry with the flag suggested above\n"+
		"unless the user has explicitly approved it. The flag bypasses a safety check\n"+
		"and the operation may be irreversible.",
		name)
}

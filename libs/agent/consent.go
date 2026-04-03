package agent

import (
	"fmt"

	"github.com/spf13/cobra"
)

// gatedFlags maps flag names to human-readable descriptions of what they do.
var gatedFlags = map[string]string{
	"force-lock":   "Override another user's active deployment lock, which may corrupt their in-progress deployment",
	"auto-approve": "Skip all confirmation prompts for destructive actions like deleting resources",
	"force":        "Bypass safety checks such as Git branch validation or remote modification detection",
}

// CheckConsentForFlags validates that an AI agent has not set any gated flags
// without explicit human approval. Returns nil if no agent is detected or if
// no gated flags are set.
func CheckConsentForFlags(cmd *cobra.Command) error {
	ctx := cmd.Context()
	if ctx == nil {
		return nil
	}

	if !isDetected(ctx) {
		return nil
	}

	for flagName, description := range gatedFlags {
		f := cmd.Flag(flagName)
		if f == nil || !f.Changed {
			continue
		}

		return fmt.Errorf(
			"the --%s flag was used by an AI agent (%s).\n\n"+
				"What this flag does: %s.\n\n"+
				"AI agents must get explicit human approval before using this flag.\n"+
				"Do not retry with this flag unless a human has reviewed and approved it.",
			flagName, Product(ctx), description)
	}

	return nil
}

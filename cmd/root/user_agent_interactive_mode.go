// This file integrates interactive mode detection with the user agent string.
//
// The detection logic is in libs/cmdio. This file retrieves the interactive
// mode from the context and adds it to the user agent.
//
// Example user agent strings:
//   - Full interactive: "cli/X.Y.Z ... interactive/full ..."
//   - Output only: "cli/X.Y.Z ... interactive/output_only ..."
//   - Non-interactive: "cli/X.Y.Z ... interactive/none ..."
package root

import (
	"context"

	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/databricks-sdk-go/useragent"
)

// Key in the user agent
const interactiveModeKey = "interactive"

func withInteractiveModeInUserAgent(ctx context.Context) context.Context {
	// mode is empty when cmdio is not initialized in the context (e.g., early startup).
	mode := cmdio.GetInteractiveMode(ctx)
	if mode == "" {
		return ctx
	}
	return useragent.InContext(ctx, interactiveModeKey, string(mode))
}

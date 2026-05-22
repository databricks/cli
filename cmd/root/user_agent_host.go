// This file integrates terminal/IDE host detection with the user agent string.
//
// The detection logic is in libs/cmdio. This file retrieves the host from
// the context and adds it to the user agent.
//
// Example user agent strings:
//   - "cli/X.Y.Z ... host/vscode ..."
//   - "cli/X.Y.Z ... host/cursor ..."
//   - "cli/X.Y.Z ... host/unknown ..."
package root

import (
	"context"

	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/databricks-sdk-go/useragent"
)

// Key in the user agent.
const hostKey = "host"

func withHostInUserAgent(ctx context.Context) context.Context {
	return useragent.InContext(ctx, hostKey, string(cmdio.DetectHost(ctx)))
}

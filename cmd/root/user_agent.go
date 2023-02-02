package root

import (
	"context"
	"strings"

	"github.com/databricks/bricks/internal/build"
	"github.com/databricks/databricks-sdk-go/useragent"
	"github.com/spf13/cobra"
)

// commandString walks up the command hierarchy of the specified
// command to build a string representing this hierarchy.
func commandString(cmd *cobra.Command) string {
	reversed := []string{cmd.Name()}
	cmd.VisitParents(func(p *cobra.Command) {
		if !p.HasParent() {
			return
		}
		reversed = append(reversed, p.Name())
	})

	ordered := make([]string, 0, len(reversed))
	for i := len(reversed) - 1; i >= 0; i-- {
		ordered = append(ordered, reversed[i])
	}

	return strings.Join(ordered, "-")
}

func withCommandInUserAgent(ctx context.Context, cmd *cobra.Command) context.Context {
	return useragent.InContext(ctx, "cmd", commandString(cmd))
}

func init() {
	useragent.WithProduct("bricks", build.GetInfo().Version)
}

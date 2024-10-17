package root

import (
	"context"
	"strings"

	"github.com/databricks/databricks-sdk-go/useragent"
	"github.com/google/uuid"
	"github.com/spf13/cobra"
)

// commandSeparator joins command names in a command hierachy.
// We enforce no command name contains this character.
// See unit test [main.TestCommandsDontUseUnderscoreInName].
const commandSeparator = "_"

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

	return strings.Join(ordered, commandSeparator)
}

func withCommandInUserAgent(ctx context.Context, cmd *cobra.Command) context.Context {
	// We add a uuid to the command context to trace multiple API requests made
	// by the same command invocation.
	newCtx := useragent.InContext(ctx, "cmd-trace-id", uuid.New().String())

	return useragent.InContext(newCtx, "cmd", commandString(cmd))
}

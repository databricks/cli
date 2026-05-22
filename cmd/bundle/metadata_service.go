package bundle

import (
	"strings"

	workspacebundle "github.com/databricks/cli/cmd/workspace/bundle"
	"github.com/spf13/cobra"
)

// metadataServiceCommands returns the auto-generated workspace bundle service
// commands keyed by their original cobra Name (e.g. "get-deployment").
//
// The auto-generated `databricks bundle <verb>` namespace collides with the
// DAB `bundle` command tree; cmd/cmd.go filters the workspace bundle root out
// of top-level registration. Here we call into the workspace package once,
// detach each subcommand from its (discarded) parent, and let the DAB bundle
// re-attach them under proper sub-groups with shorter names.
func metadataServiceCommands() map[string]*cobra.Command {
	ws := workspacebundle.New()
	subs := ws.Commands()
	out := make(map[string]*cobra.Command, len(subs))
	for _, sub := range subs {
		ws.RemoveCommand(sub)
		// These ride under the DAB bundle now, which is visible.
		sub.Hidden = false
		out[sub.Name()] = sub
	}
	return out
}

// renameTo replaces the first whitespace-separated token of cobra's Use field
// with newName, preserving the trailing positional-arg syntax that cobra renders
// in usage strings (e.g. "get-deployment NAME" -> "get NAME"). Returns the
// command for inline chaining.
func renameTo(c *cobra.Command, newName string) *cobra.Command {
	rest := ""
	if i := strings.IndexByte(c.Use, ' '); i >= 0 {
		rest = c.Use[i:]
	}
	c.Use = newName + rest
	return c
}

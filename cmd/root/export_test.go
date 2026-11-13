package root

import (
	"context"

	"github.com/spf13/cobra"
)

// BuildAll builds every command in t and returns the root command, for tests
// that walk the whole tree.
func BuildAll(ctx context.Context, t *Tree) *cobra.Command {
	return t.buildAll(ctx)
}

package pipelines

import (
	"context"

	"github.com/spf13/cobra"
)

func New(ctx context.Context) *cobra.Command {
	cli := NewRoot(ctx)
	cli.AddCommand(initCommand())
	return cli
}

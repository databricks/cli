package pipelines

import (
	"context"

	"github.com/databricks/cli/cmd/pipelines/root"
	"github.com/spf13/cobra"
)

func New(ctx context.Context) *cobra.Command {
	cli := root.New(ctx)
	cli.AddCommand(initCommand())
	return cli
}

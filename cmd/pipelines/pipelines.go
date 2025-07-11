package pipelines

import (
	"context"

	"github.com/databricks/cli/cmd/pipelines/root"
	"github.com/spf13/cobra"
)

func New(ctx context.Context) *cobra.Command {
	cli := root.New(ctx)
	initVariableFlag(cli)
	cli.AddCommand(initCommand())
	cli.AddCommand(openCommand())
	cli.AddCommand(deployCommand())
	cli.AddCommand(runCommand())
	cli.AddCommand(dryRunCommand())
	cli.AddCommand(authCommand())
	cli.AddCommand(destroyCommand())
	cli.AddCommand(versionCommand())
	return cli
}

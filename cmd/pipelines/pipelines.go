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
	cli.AddCommand(StopCommand(StopCommandOpts{SkipInitContext: false}))
	cli.AddCommand(historyCommand())
	cli.AddCommand(logsCommand())
	cli.AddCommand(versionCommand())
	return cli
}

// ManagementGroupID contains auto-generated CLI commands for Pipelines API,
// that are separate from main CLI commands defined in Commands.
const ManagementGroupID = "management"

// Commands returns the list of commands that are shared between
// the standalone pipelines CLI and databricks pipelines.
// Note: auth and version are excluded as they are only for standalone CLI.
func Commands() []*cobra.Command {
	return []*cobra.Command{
		initCommand(),
		deployCommand(),
		destroyCommand(),
		runCommand(),
		dryRunCommand(),
		historyCommand(),
		logsCommand(),
		openCommand(),
	}
}

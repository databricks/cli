package cmd

import (
	"github.com/databricks/cli/cmd/account"
	"github.com/databricks/cli/cmd/api"
	"github.com/databricks/cli/cmd/auth"
	"github.com/databricks/cli/cmd/bundle"
	"github.com/databricks/cli/cmd/configure"
	"github.com/databricks/cli/cmd/fs"
	"github.com/databricks/cli/cmd/labs"
	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/cmd/sync"
	"github.com/databricks/cli/cmd/version"
	"github.com/databricks/cli/cmd/workspace"
	"github.com/spf13/cobra"
)

func New() *cobra.Command {
	cli := root.New()

	// Add account subcommand.
	cli.AddCommand(account.New())

	// Add workspace subcommands.
	for _, cmd := range workspace.All() {
		cli.AddCommand(cmd)
	}

	// Add workspace command groups.
	groups := workspace.Groups()
	for i := range groups {
		cli.AddGroup(&groups[i])
	}

	// Add other subcommands.
	cli.AddCommand(api.New())
	cli.AddCommand(auth.New())
	cli.AddCommand(bundle.New())
	cli.AddCommand(configure.New())
	cli.AddCommand(fs.New())
	cli.AddCommand(labs.New())
	cli.AddCommand(sync.New())
	cli.AddCommand(version.New())

	return cli
}

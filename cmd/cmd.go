package cmd

import (
	"sync"

	"github.com/databricks/cli/cmd/account"
	"github.com/databricks/cli/cmd/api"
	"github.com/databricks/cli/cmd/auth"
	"github.com/databricks/cli/cmd/bundle"
	"github.com/databricks/cli/cmd/configure"
	"github.com/databricks/cli/cmd/fs"
	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/cmd/workspace"
	"github.com/spf13/cobra"
)

var once sync.Once
var cmd *cobra.Command

func New() *cobra.Command {
	// TODO: this command is still a global.
	// Once the non-generated commands are all instantiatable,
	// we can remove the global and instantiate this as well.
	once.Do(func() {
		cli := root.RootCmd

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

		cmd = cli
	})

	return cmd
}

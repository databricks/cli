package cmd

import (
	"context"
	"strings"

	"github.com/databricks/cli/cmd/account"
	"github.com/databricks/cli/cmd/api"
	"github.com/databricks/cli/cmd/auth"
	"github.com/databricks/cli/cmd/bundle"
	"github.com/databricks/cli/cmd/configure"
	"github.com/databricks/cli/cmd/fs"
	"github.com/databricks/cli/cmd/labs"
	"github.com/databricks/cli/cmd/openai"
	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/cmd/sync"
	"github.com/databricks/cli/cmd/version"
	"github.com/databricks/cli/cmd/workspace"
	"github.com/spf13/cobra"
)

const (
	mainGroup        = "main"
	permissionsGroup = "permissions"
)

func New(ctx context.Context) *cobra.Command {
	cli := root.New(ctx)

	// Add account subcommand.
	cli.AddCommand(account.New())

	// Add workspace subcommands.
	for _, cmd := range workspace.All() {
		// Built-in groups for the workspace commands.
		groups := []cobra.Group{
			{
				ID:    mainGroup,
				Title: "Available Commands",
			},
			{
				ID:    permissionsGroup,
				Title: "Permission Commands",
			},
		}
		for i := range groups {
			cmd.AddGroup(&groups[i])
		}

		// Order the permissions subcommands after the main commands.
		for _, sub := range cmd.Commands() {
			switch {
			case strings.HasSuffix(sub.Name(), "-permissions"), strings.HasSuffix(sub.Name(), "-permission-levels"):
				sub.GroupID = permissionsGroup
			default:
				sub.GroupID = mainGroup
			}
		}

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
	cli.AddCommand(labs.New(ctx))
	cli.AddCommand(sync.New())
	cli.AddCommand(version.New())
	cli.AddCommand(openai.New())

	return cli
}

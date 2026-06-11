package cmd

import (
	"context"
	"strings"

	aitoolscmd "github.com/databricks/cli/cmd/aitools"
	"github.com/databricks/cli/cmd/psql"
	ssh "github.com/databricks/cli/experimental/ssh/cmd"

	"github.com/databricks/cli/cmd/account"
	"github.com/databricks/cli/cmd/api"
	"github.com/databricks/cli/cmd/auth"
	"github.com/databricks/cli/cmd/bundle"
	"github.com/databricks/cli/cmd/cache"
	"github.com/databricks/cli/cmd/completion"
	"github.com/databricks/cli/cmd/configure"
	"github.com/databricks/cli/cmd/experimental"
	"github.com/databricks/cli/cmd/fs"
	"github.com/databricks/cli/cmd/labs"
	"github.com/databricks/cli/cmd/pipelines"
	"github.com/databricks/cli/cmd/quickstart"
	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/cmd/sandbox"
	"github.com/databricks/cli/cmd/selftest"
	"github.com/databricks/cli/cmd/sync"
	"github.com/databricks/cli/cmd/version"
	"github.com/databricks/cli/cmd/workspace"
	"github.com/databricks/cli/libs/cmdgroup"
	"github.com/spf13/cobra"
)

const (
	mainGroup        = "main"
	permissionsGroup = "permissions"
)

// configureGroups adds groups to the command, only if a group
// has at least one available command. When only one group survives
// filtering, the grouping is dropped so Cobra's default "Available
// Commands" heading is used — matching commands that don't define
// groups at all.
func configureGroups(cmd *cobra.Command, groups []cobra.Group) {
	filteredGroups := cmdgroup.FilterGroups(groups, cmd.Commands())
	if len(filteredGroups) <= 1 {
		for _, sub := range cmd.Commands() {
			sub.GroupID = ""
		}
		return
	}
	for i := range filteredGroups {
		cmd.AddGroup(&filteredGroups[i])
	}
}

func accountCommand() *cobra.Command {
	cmd := account.New()
	configureGroups(cmd, account.Groups())
	return cmd
}

func New(ctx context.Context) *cobra.Command {
	cli := root.New(ctx)

	// Add account subcommand.
	cli.AddCommand(accountCommand())

	// Add workspace subcommands.
	workspaceCommands := workspace.All()
	for _, cmd := range workspaceCommands {
		// The auto-generated `bundle` workspace service (DMS) shares its name
		// with the DAB `bundle` command tree (cmd/bundle). Registering both
		// here clobbers the DAB tree's help output. Skip the generated one;
		// callers still have `databricks api ...` for the DMS endpoints.
		if cmd.Name() == "bundle" {
			continue
		}
		// Order the permissions subcommands after the main commands.
		for _, sub := range cmd.Commands() {
			// some commands override groups in overrides.go, leave them as-is
			if sub.GroupID != "" {
				continue
			}

			switch {
			case strings.HasSuffix(sub.Name(), "-permissions"), strings.HasSuffix(sub.Name(), "-permission-levels"):
				sub.GroupID = permissionsGroup
			default:
				sub.GroupID = mainGroup
			}
		}

		cli.AddCommand(cmd)

		// Built-in groups for the workspace commands.
		groups := []cobra.Group{
			{
				ID:    mainGroup,
				Title: "Main Commands",
			},
			{
				ID:    pipelines.ManagementGroupID,
				Title: "Management Commands",
			},
			{
				ID:    permissionsGroup,
				Title: "Permission Commands",
			},
		}

		configureGroups(cmd, groups)
	}

	// Add other subcommands.
	cli.AddCommand(aitoolscmd.NewAitoolsCmd())
	cli.AddCommand(api.New())
	cli.AddCommand(auth.New())
	cli.AddCommand(completion.New())
	cli.AddCommand(bundle.New())
	cli.AddCommand(cache.New())
	cli.AddCommand(experimental.New())
	cli.AddCommand(psql.New())
	cli.AddCommand(configure.New())
	cli.AddCommand(fs.New())
	cli.AddCommand(labs.New(ctx))
	cli.AddCommand(sandbox.New())
	cli.AddCommand(sync.New())
	cli.AddCommand(version.New())
	cli.AddCommand(quickstart.New())
	cli.AddCommand(selftest.New())
	cli.AddCommand(ssh.New())

	// Add workspace command groups, filtering out empty groups or groups with only hidden commands.
	configureGroups(cli, append(workspace.Groups(), cobra.Group{
		ID:    "development",
		Title: "Developer Tools",
	}))

	return cli
}

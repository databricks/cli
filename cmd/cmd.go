package cmd

import (
	"context"
	"strings"

	"github.com/databricks/cli/cmd/psql"

	"github.com/databricks/cli/cmd/account"
	"github.com/databricks/cli/cmd/api"
	"github.com/databricks/cli/cmd/auth"
	"github.com/databricks/cli/cmd/bundle"
	"github.com/databricks/cli/cmd/configure"
	"github.com/databricks/cli/cmd/fs"
	"github.com/databricks/cli/cmd/labs"
	"github.com/databricks/cli/cmd/pipelines"
	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/cmd/selftest"
	"github.com/databricks/cli/cmd/sync"
	"github.com/databricks/cli/cmd/version"
	"github.com/databricks/cli/cmd/workspace"
	"github.com/spf13/cobra"
)

const (
	mainGroup        = "main"
	permissionsGroup = "permissions"
)

// filterGroups returns command groups that have at least one available (non-hidden) command.
// Empty groups or groups with only hidden commands are filtered out from the help output.
// Commands that belong to filtered groups will have their GroupID cleared.
func filterGroups(groups []cobra.Group, allCommands []*cobra.Command) []cobra.Group {
	var filteredGroups []cobra.Group

	// Create a map to track which groups have available commands
	groupHasAvailableCommands := make(map[string]bool)

	// Check each command to see if it belongs to a group and is available
	for _, cmd := range allCommands {
		if cmd.GroupID != "" && cmd.IsAvailableCommand() {
			groupHasAvailableCommands[cmd.GroupID] = true
		}
	}

	// Collect groups that have available commands
	validGroupIDs := make(map[string]bool)
	for _, group := range groups {
		if groupHasAvailableCommands[group.ID] {
			filteredGroups = append(filteredGroups, group)
			validGroupIDs[group.ID] = true
		}
	}

	// Clear GroupID for commands that belong to filtered groups
	for _, cmd := range allCommands {
		if cmd.GroupID != "" && !validGroupIDs[cmd.GroupID] {
			cmd.GroupID = ""
		}
	}

	return filteredGroups
}

func New(ctx context.Context) *cobra.Command {
	cli := root.New(ctx)

	// Add account subcommand.
	cli.AddCommand(account.New())

	// Add workspace subcommands.
	workspaceCommands := workspace.All()
	for _, cmd := range workspaceCommands {
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

	// Add other subcommands.
	cli.AddCommand(api.New())
	cli.AddCommand(auth.New())
	cli.AddCommand(bundle.New())
	cli.AddCommand(psql.New())
	cli.AddCommand(configure.New())
	cli.AddCommand(fs.New())
	cli.AddCommand(labs.New(ctx))
	cli.AddCommand(sync.New())
	cli.AddCommand(version.New())
	cli.AddCommand(selftest.New())
	cli.AddCommand(pipelines.InstallPipelinesCLI())

	// Add workspace command groups, filtering out empty groups or groups with only hidden commands.
	allGroups := workspace.Groups()
	allCommands := cli.Commands()
	filteredGroups := filterGroups(allGroups, allCommands)
	for i := range filteredGroups {
		cli.AddGroup(&filteredGroups[i])
	}

	return cli
}

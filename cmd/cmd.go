package cmd

import (
	"context"
	"strings"

	aircmd "github.com/databricks/cli/cmd/air"
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

// withGroups returns spec with a Build that also groups the command's
// subcommands into groups, keeping only groups that have available commands.
func withGroups(spec root.CommandSpec, groups []cobra.Group) root.CommandSpec {
	build := spec.Build
	spec.Build = func(ctx context.Context, cmd *cobra.Command) {
		build(ctx, cmd)
		cmdgroup.Configure(cmd, groups)
	}
	return spec
}

// workspaceSpec returns spec with a Build that also groups the subcommands of
// a generated workspace command and orders its permissions subcommands last.
func workspaceSpec(spec root.CommandSpec) root.CommandSpec {
	build := spec.Build
	spec.Build = func(ctx context.Context, cmd *cobra.Command) {
		build(ctx, cmd)

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
	}
	return withGroups(spec, []cobra.Group{
		{ID: mainGroup, Title: "Main Commands"},
		{ID: pipelines.ManagementGroupID, Title: "Management Commands"},
		{ID: permissionsGroup, Title: "Permission Commands"},
	})
}

// New returns the CLI's command tree.
func New(ctx context.Context) *root.Tree {
	cli := root.NewTree(ctx)

	specs := []root.CommandSpec{
		withGroups(account.Spec, account.Groups()),
		aircmd.Spec,
		aitoolscmd.Spec,
		api.Spec,
		auth.Spec,
		completion.Spec,
		bundle.Spec,
		cache.Spec,
		experimental.Spec,
		psql.Spec,
		configure.Spec,
		fs.Spec,
		labs.Spec,
		sandbox.Spec,
		sync.Spec,
		version.Spec,
		quickstart.Spec,
		selftest.Spec,
		ssh.Spec,
	}
	// Hand-written commands that replace a generated workspace command of the
	// same name. Cobra resolves duplicate names by position after an unstable
	// sort, so the generated command must not be added at all.
	name := func(spec root.CommandSpec) string { return strings.Fields(spec.Use)[0] }
	replaced := map[string]bool{
		name(sandbox.Spec): true,
	}
	for _, spec := range workspace.All() {
		if !replaced[name(spec)] {
			specs = append(specs, workspaceSpec(spec))
		}
	}
	cli.Add(specs...)

	// Add workspace command groups, filtering out empty groups or groups with only hidden commands.
	cli.AddGroups(append(workspace.Groups(), cobra.Group{
		ID:    "development",
		Title: "Developer Tools",
	})...)

	return cli
}

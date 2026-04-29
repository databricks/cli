// Package deployment wires the `databricks ucm deployment` subcommand group:
// `bind` attaches a ucm-declared resource to an existing UC object; `unbind`
// drops that recorded binding. Mirrors cmd/bundle/deployment in shape, but
// forks rather than imports so the bundle package can evolve upstream
// independently of ucm.
package deployment

import (
	"github.com/spf13/cobra"
)

// New returns the cobra group registered under `databricks ucm deployment`.
func New() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "deployment",
		Short: "Deployment related commands",
		Long: `Deployment related commands for managing ucm resource bindings.

Use these commands to bind / unbind ucm definitions to existing Unity Catalog
objects so that subsequent deploys update — rather than recreate — them.

Common workflow:
1. Author a ucm.yml that declares the resource with the desired target state.

2. Bind the ucm key to the existing Unity Catalog object:
   databricks ucm deployment bind my_catalog team_alpha

3. Deploy updates — the bound resource is reconciled in place:
   databricks ucm deploy`,
	}

	cmd.AddCommand(newBindCommand())
	cmd.AddCommand(newMigrateCommand())
	cmd.AddCommand(newUnbindCommand())
	return cmd
}

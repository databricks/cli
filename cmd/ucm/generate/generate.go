// Package generate implements the per-kind subcommands of `databricks ucm
// generate`. Mirrors cmd/bundle/generate's shape: one cobra subcommand per UC
// resource kind, each fetching one resource by name and emitting a per-resource
// YAML fragment that can be added to a ucm.yml project.
package generate

import (
	"github.com/spf13/cobra"
)

// New returns the `ucm generate` parent cobra command. Subcommands are
// registered per UC resource kind; each is a brownfield import for one
// named resource into ucm.yml. The shared --key flag lives on the parent
// so each subcommand can read it via cmd.Flag("key") (cobra walks the
// parent chain).
func New() *cobra.Command {
	var key string

	cmd := &cobra.Command{
		Use:   "generate",
		Short: "Generate ucm configuration from existing Unity Catalog resources",
		Long: `Generate ucm configuration from existing Unity Catalog resources.

Common patterns:
  databricks ucm generate catalog --existing-catalog-name prod --key prod_catalog
  databricks ucm generate schema --existing-schema-name prod.events --key events_schema
  databricks ucm generate volume --existing-volume-name prod.raw.landing --key landing_volume
  databricks ucm generate external-location --existing-external-location-name prod_loc --key prod_loc
  databricks ucm generate storage-credential --existing-storage-credential-name prod_cred --key prod_cred
  databricks ucm generate connection --existing-connection-name prod_conn --key prod_conn

Use --key to specify the resource key in your ucm.yml configuration. If
--key is omitted, the resource name is sanitized (dots replaced with
underscores) into a valid key.`,
	}

	cmd.AddCommand(NewGenerateCatalogCommand())
	cmd.AddCommand(NewGenerateSchemaCommand())
	cmd.AddCommand(NewGenerateVolumeCommand())
	cmd.AddCommand(NewGenerateExternalLocationCommand())
	cmd.AddCommand(NewGenerateStorageCredentialCommand())
	cmd.AddCommand(NewGenerateConnectionCommand())
	cmd.PersistentFlags().StringVar(&key, "key", "", `resource key to use for the generated configuration`)
	return cmd
}

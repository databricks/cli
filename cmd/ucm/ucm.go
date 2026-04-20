// Package ucm implements the `databricks ucm` subcommand for managing Unity Catalog
// resources (metastores, catalogs, schemas, external locations, storage credentials,
// grants, tags, connections) and their cloud-underlay prerequisites at enterprise scale.
//
// It mirrors the UX and engine of `databricks bundle` (Declarative Automation Bundles)
// but targets Unity Catalog declarative configuration instead of jobs/pipelines.
package ucm

import (
	"github.com/spf13/cobra"
)

func New() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "ucm",
		Short: "Manage Unity Catalog resources declaratively at enterprise scale.",
		Long: `Unity Catalog Management (ucm) — DAB-style declarative management of Unity Catalog.

Common workflows:
  databricks ucm init greenfield-bootstrap    # Scaffold a new ucm.yml project
  databricks ucm validate                     # Lint config + run policy checks
  databricks ucm plan --target dev            # Preview changes
  databricks ucm deploy --target prod         # Apply changes
  databricks ucm destroy --target dev         # Tear down a target

Import existing resources:
  databricks ucm generate --metastore-name m1  # Scan an account + emit ucm.yml
  databricks ucm import catalog team_alpha     # Import a single catalog into state

Governance:
  databricks ucm drift --target prod           # Detect out-of-band changes
  databricks ucm policy-check                  # Run only the validation mutators

Online documentation: https://docs.databricks.com/en/dev-tools/ucm/index.html`,
		GroupID: "development",
	}

	cmd.AddCommand(newValidateCommand())
	cmd.AddCommand(newSchemaCommand())
	cmd.AddCommand(newPlanCommand())
	cmd.AddCommand(newDeployCommand())
	cmd.AddCommand(newDestroyCommand())
	cmd.AddCommand(newSummaryCommand())
	cmd.AddCommand(newInitCommand())
	cmd.AddCommand(newGenerateCommand())
	cmd.AddCommand(newBindCommand())
	cmd.AddCommand(newDebugCommand())
	cmd.AddCommand(newDiffCommand())
	cmd.AddCommand(newDriftCommand())
	cmd.AddCommand(newImportCommand())
	cmd.AddCommand(newPolicyCheckCommand())
	return cmd
}

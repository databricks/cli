package bundle

import (
	"github.com/databricks/cli/cmd/bundle/deployment"
	"github.com/spf13/cobra"
)

func New() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "bundle",
		Short: "Declarative Automation Bundles let you express data/AI/analytics projects as code.",
		Long: `Declarative Automation Bundles let you express data/AI/analytics projects as code.

Common workflows:
  databricks bundle init default-python      # Initialize new project
  databricks bundle deploy --target dev      # Deploy to development
  databricks bundle run my_job               # Run jobs/pipelines
  databricks bundle deploy --target prod     # Deploy to production

Import existing resources:
  databricks bundle generate job --existing-job-id 123 --key my_job # Generate job configuration
  databricks bundle deployment bind my_job 123                      # Link to an existing job

Online documentation: https://docs.databricks.com/en/dev-tools/bundles/index.html`,
		GroupID: "development",
	}

	initVariableFlag(cmd)
	cmd.AddCommand(newDeployCommand())
	cmd.AddCommand(newDestroyCommand())
	cmd.AddCommand(newRunCommand())
	cmd.AddCommand(newSchemaCommand())
	cmd.AddCommand(newSyncCommand())
	cmd.AddCommand(newValidateCommand())
	cmd.AddCommand(newInitCommand())
	cmd.AddCommand(newSummaryCommand())
	cmd.AddCommand(newGenerateCommand())
	cmd.AddCommand(newDebugCommand())
	cmd.AddCommand(newOpenCommand())
	cmd.AddCommand(newPlanCommand())
	cmd.AddCommand(newConfigRemoteSyncCommand())

	// Bundle Metadata Service (DMS) read-only command groups. Only `get`
	// and `list` are surfaced here; mutating verbs (create/delete/heartbeat/
	// complete) are not user-facing yet and stay in the auto-generated
	// `cmd/workspace/bundle` tree (which is filtered out of top-level
	// registration in cmd/cmd.go).
	//
	// Each group uses the singular noun as the primary `Use` and aliases the
	// plural for discoverability; cobra renders the alias pair in the help
	// header so callers see both forms.
	dms := metadataServiceCommands()

	// The DAB `deployment` group already exists for bind/unbind/migrate.
	// Augment it additively with the DMS read-side verbs and add the
	// `deployments` alias.
	deploymentCmd := deployment.NewDeploymentCommand()
	deploymentCmd.Aliases = append(deploymentCmd.Aliases, "deployments")
	deploymentCmd.AddCommand(renameTo(dms["get-deployment"], "get"))
	deploymentCmd.AddCommand(renameTo(dms["list-deployments"], "list"))
	cmd.AddCommand(deploymentCmd)

	versionCmd := &cobra.Command{
		Use:     "version",
		Aliases: []string{"versions"},
		Short:   "Read version records in the bundle metadata service.",
	}
	versionCmd.AddCommand(renameTo(dms["get-version"], "get"))
	versionCmd.AddCommand(renameTo(dms["list-versions"], "list"))
	cmd.AddCommand(versionCmd)

	resourceCmd := &cobra.Command{
		Use:     "resource",
		Aliases: []string{"resources"},
		Short:   "Read resource records from the bundle metadata service.",
	}
	resourceCmd.AddCommand(renameTo(dms["get-resource"], "get"))
	resourceCmd.AddCommand(renameTo(dms["list-resources"], "list"))
	cmd.AddCommand(resourceCmd)

	operationCmd := &cobra.Command{
		Use:     "operation",
		Aliases: []string{"operations"},
		Short:   "Read operation records in the bundle metadata service.",
	}
	operationCmd.AddCommand(renameTo(dms["get-operation"], "get"))
	operationCmd.AddCommand(renameTo(dms["list-operations"], "list"))
	cmd.AddCommand(operationCmd)

	return cmd
}

package deployment

import (
	"fmt"
	"os"

	"github.com/databricks/cli/bundle/deploy/terraform"
	"github.com/databricks/cli/bundle/direct"
	"github.com/databricks/cli/bundle/direct/dstate"
	"github.com/databricks/cli/cmd/bundle/utils"
	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/logdiag"
	"github.com/spf13/cobra"
)

func newMigrateCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "migrate",
		Short: "Migrate from Terraform to Direct deployment engine",
		Long: `This command converts your bundle from using Terraform for deployment to using
the Direct deployment engine. It reads resource IDs from the existing Terraform
state and creates a Direct deployment state file (resources.json) with the same
lineage and incremented serial number.

Note, the migration is performed locally only. To finalize it, run 'bundle deploy'. This will synchronize the state file
to the workspace so that subsequent deploys of this bundle use direct deployment engine as well.

WARNING: Both direct deployment engine and this command are experimental and not recommended for production targets yet.
`,
		Args: root.NoArgs,
	}

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		opts := utils.ProcessOptions{
			SkipEngineEnvVar: true,
			AlwaysPull:       true,
			FastValidate:     true,
			Build:            true,
		}

		b, stateDesc, err := utils.ProcessBundleRet(cmd, opts)
		if err != nil {
			return err
		}
		ctx := cmd.Context()

		if stateDesc.Lineage == "" {
			cmdio.LogString(ctx, `This command migrates the existing Terraform state file (terraform.tfstate) to a direct deployment state file (resources.json). However, no existing local or remote state was found.

To start using direct engine, deploy with DATABRICKS_BUNDLE_ENGINE=direct env var set.`) // TODO: mention bundle.engine once it's there
			return root.ErrAlreadyPrinted
		}

		if stateDesc.Engine.IsDirect() {
			return fmt.Errorf("already using direct engine\nDetails: %s", stateDesc.String())
		}

		_, localTerraformPath := b.StateFilenameTerraform(ctx)
		if _, err = os.Stat(localTerraformPath); err != nil {
			return fmt.Errorf("reading %s: %w", localTerraformPath, err)
		}

		terraformResources, err := terraform.ParseResourcesState(ctx, b)
		if err != nil {
			return fmt.Errorf("failed to parse terraform state: %w", err)
		}

		_, localPath := b.StateFilenameDirect(ctx)
		tempStatePath := localPath + ".temp-migration"
		if _, err = os.Stat(tempStatePath); err == nil {
			return fmt.Errorf("temporary state file %s already exists, another migration is in progress or was interrupated. In the latter case, delete the file", tempStatePath)
		}
		if _, err = os.Stat(localPath); err == nil {
			return fmt.Errorf("state file %s already exists", localPath)
		}

		state := make(map[string]dstate.ResourceEntry)
		for groupName, group := range terraformResources {
			for key, resourceEntry := range group {
				newKey := fmt.Sprintf("resources.%s.%s", groupName, key)
				state[newKey] = dstate.ResourceEntry{ID: resourceEntry.ID}
			}
		}

		deploymentBundle := &direct.DeploymentBundle{
			StateDB: dstate.DeploymentState{
				Path: tempStatePath,
				Data: dstate.Database{
					Serial:  stateDesc.Serial + 1,
					Lineage: stateDesc.Lineage,
					State:   state,
				},
			},
		}

		plan, err := deploymentBundle.CalculatePlan(ctx, b.WorkspaceClient(), &b.Config, tempStatePath)
		if err != nil {
			return err
		}

		deploymentBundle.Apply(ctx, b.WorkspaceClient(), &b.Config, plan, direct.MigrateMode(true))

		if err := os.Rename(tempStatePath, localPath); err != nil {
			return fmt.Errorf("renaming %s to %s: %w", tempStatePath, localPath, err)
		}
		err = os.Remove(localTerraformPath)
		if err != nil {
			// not fatal, since we've increased serial
			logdiag.LogError(ctx, err)
		}

		cmdio.LogString(ctx, fmt.Sprintf(`Migrated %d resources to direct engine state file: %s
To finalize deployment, run "bundle deploy".`, len(deploymentBundle.StateDB.Data.State), localPath))
		return nil
	}

	return cmd
}

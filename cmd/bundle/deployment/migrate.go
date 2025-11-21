package deployment

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/deploy"
	"github.com/databricks/cli/bundle/deploy/terraform"
	"github.com/databricks/cli/bundle/direct"
	"github.com/databricks/cli/bundle/direct/dstate"
	"github.com/databricks/cli/bundle/statemgmt"
	"github.com/databricks/cli/cmd/bundle/utils"
	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/logdiag"
	"github.com/databricks/cli/libs/structs/structaccess"
	"github.com/databricks/cli/libs/structs/structpath"
	"github.com/spf13/cobra"
)

const backupSuffix = ".backup"

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
			// Same options as regular deploy, to ensure bundle config is in the same state
			FastValidate: true,
			Build:        true,
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
			return fmt.Errorf("temporary state file %s already exists, another migration is in progress or was interrupted. In the latter case, delete the file", tempStatePath)
		}
		if _, err = os.Stat(localPath); err == nil {
			return fmt.Errorf("state file %s already exists", localPath)
		}

		etags := map[string]string{}

		state := make(map[string]dstate.ResourceEntry)
		for key, resourceEntry := range terraformResources {
			state[key] = dstate.ResourceEntry{ID: resourceEntry.ID}
			if resourceEntry.ETag != "" {
				// dashboard:
				etags[key] = resourceEntry.ETag
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

		for key, planEntry := range plan.Plan {
			etag := etags[key]
			if etag == "" {
				continue
			}
			err := structaccess.Set(planEntry.NewState.Value, structpath.NewStringKey(nil, "etag"), etag)
			if err != nil {
				return fmt.Errorf("failed to set etag on %s: %w", key, err)
			}
		}

		deploymentBundle.Apply(ctx, b.WorkspaceClient(), &b.Config, plan, direct.MigrateMode(true))
		if logdiag.HasError(ctx) {
			logdiag.LogError(ctx, errors.New("migration failed; ensure you have done full deploy before the migration"))
			return root.ErrAlreadyPrinted
		}

		if err := os.Rename(tempStatePath, localPath); err != nil {
			return fmt.Errorf("renaming %s to %s: %w", tempStatePath, localPath, err)
		}

		localTerraformBackupPath := localTerraformPath + backupSuffix

		err = os.Rename(localTerraformPath, localTerraformBackupPath)
		if err != nil {
			// not fatal, since we've increased serial
			logdiag.LogError(ctx, err)
		}

		err = backupRemoteTerraformState(ctx, b, stateDesc)
		if err != nil {
			logdiag.LogError(ctx, err)
		}

		cmdio.LogString(ctx, fmt.Sprintf(`Migrated %d resources to direct engine state file: %s

Validate the migration by running "bundle plan", there should be no actions planned.

The state file is not synchronized to the workspace yet. To do that and finalize the migration, run "bundle deploy".

To undo the migration, remove %s and rename %s to %s
`, len(deploymentBundle.StateDB.Data.State), localPath, localPath, localTerraformBackupPath, localTerraformPath))
		return nil
	}

	return cmd
}

func findRemoteTerraformState(states []*statemgmt.StateDesc) *statemgmt.StateDesc {
	for _, st := range states {
		if !st.Engine.IsDirect() && !st.IsLocal {
			return st
		}
	}

	return nil
}

func backupRemoteTerraformState(ctx context.Context, b *bundle.Bundle, winner *statemgmt.StateDesc) error {
	remoteTF := findRemoteTerraformState(winner.AllStates)
	if remoteTF == nil {
		return nil
	}

	filer, err := deploy.StateFiler(b)
	if err != nil {
		return err
	}

	err = filer.Write(ctx, remoteTF.SourcePath+backupSuffix, bytes.NewReader(remoteTF.Content))
	if err != nil {
		return fmt.Errorf("saving backup of TF state to %q: %w", remoteTF.SourcePath, err)
	}

	err = filer.Delete(ctx, remoteTF.SourcePath)
	if err != nil {
		return fmt.Errorf("deleting remote tf state at %q: %w", remoteTF.SourcePath, err)
	}

	return nil
}

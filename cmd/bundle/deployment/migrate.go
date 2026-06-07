package deployment

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config/engine"
	"github.com/databricks/cli/bundle/config/mutator/resourcemutator"
	"github.com/databricks/cli/bundle/direct/dresources"
	"github.com/databricks/cli/bundle/direct/dstate"
	"github.com/databricks/cli/bundle/migrate"
	"github.com/databricks/cli/cmd/bundle/utils"
	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/env"
	"github.com/databricks/cli/libs/logdiag"
	"github.com/databricks/cli/libs/shellquote"
	"github.com/spf13/cobra"
)

const backupSuffix = ".backup"

func getCommonArgs(cmd *cobra.Command) string {
	var quotedArgs []string

	if flag := cmd.Flag("target"); flag != nil && flag.Changed {
		if target := flag.Value.String(); target != "" {
			quotedArgs = append(quotedArgs, "-t", shellquote.BashArg(target))
		}
	}
	if flag := cmd.Flag("profile"); flag != nil && flag.Changed {
		if profile := flag.Value.String(); profile != "" {
			quotedArgs = append(quotedArgs, "-p", shellquote.BashArg(profile))
		}
	}
	if flag := cmd.Flag("var"); flag != nil && flag.Changed {
		if varValues, err := cmd.Flags().GetStringSlice("var"); err == nil {
			for _, v := range varValues {
				quotedArgs = append(quotedArgs, "--var", shellquote.BashArg(v))
			}
		}
	}

	if len(quotedArgs) == 0 {
		return ""
	}
	return " " + strings.Join(quotedArgs, " ")
}

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
`,
		Args: root.NoArgs,
	}

	// --noplancheck kept for backward compatibility; the plan check was removed
	// because the command no longer invokes the Terraform engine.
	cmd.Flags().Bool("noplancheck", false, "No-op (kept for compatibility).")

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		extraArgsStr := getCommonArgs(cmd)

		// Clear the engine env var so migrate always uses terraform engine to read existing state,
		// regardless of what the user may have set in their environment.
		cmd.SetContext(env.Set(cmd.Context(), engine.EnvVar, ""))

		opts := utils.ProcessOptions{
			AlwaysPull: true,
			// Same options as regular deploy, to ensure bundle config is in the same state
			FastValidate: true,
			Build:        true,
			PostInitFunc: func(_ context.Context, b *bundle.Bundle) error {
				if b.Config.Bundle.Engine == engine.EngineTerraform {
					return fmt.Errorf("bundle.engine is set to %q. Migration requires \"engine: direct\" or no engine setting. Change the setting to \"engine: direct\" and retry", engine.EngineTerraform)
				}
				return nil
			},
		}

		b, stateDesc, err := utils.ProcessBundleRet(cmd, opts)
		if err != nil {
			return err
		}
		ctx := cmd.Context()

		if stateDesc.Lineage == "" {
			cmdio.LogString(ctx, `Error: This command migrates the existing Terraform state file (terraform.tfstate) to a direct deployment state file (resources.json). However, no existing local or remote state was found.

To start using direct engine, set "engine: direct" under bundle in your databricks.yml or deploy with DATABRICKS_BUNDLE_ENGINE=direct env var set.`)
			return root.ErrAlreadyPrinted
		}

		if stateDesc.Engine.IsDirect() {
			return fmt.Errorf("already using direct engine\nDetails: %s", stateDesc.String())
		}

		_, localTerraformPath := b.StateFilenameTerraform(ctx)
		if _, err = os.Stat(localTerraformPath); err != nil {
			return fmt.Errorf("reading %s: %w", localTerraformPath, err)
		}

		tfAttrs, terraformResources, _, err := migrate.ParseTFStateFull(ctx, localTerraformPath)
		if err != nil {
			return fmt.Errorf("failed to parse terraform state: %w", err)
		}

		for key, resourceEntry := range terraformResources {
			if resourceEntry.ID == "" {
				return fmt.Errorf("failed to intepret terraform state for %s: missing ID", key)
			}
		}

		_, localPath := b.StateFilenameDirect(ctx)
		tempStatePath := localPath + ".temp-migration"
		if _, err = os.Stat(tempStatePath); err == nil {
			return fmt.Errorf("temporary state file %s already exists, another migration is in progress or was interrupted. In the latter case, delete the file", tempStatePath)
		}
		if _, err = os.Stat(localPath); err == nil {
			return fmt.Errorf("state file %s already exists", localPath)
		}

		state := make(map[string]dstate.ResourceEntry)
		for key, resourceEntry := range terraformResources {
			state[key] = dstate.ResourceEntry{
				ID:    resourceEntry.ID,
				State: json.RawMessage("{}"),
			}
		}

		migratedDB := dstate.NewDatabase(stateDesc.Lineage, stateDesc.Serial+1)
		migratedDB.State = state

		var stateDB dstate.DeploymentState
		stateDB.OpenWithData(tempStatePath, migratedDB)

		tempStatePathAutoRemove := true

		defer func() {
			if tempStatePathAutoRemove {
				_ = os.Remove(tempStatePath)
			}
		}()

		// Apply SecretScopeFixups so the config matches what the direct engine expects.
		// This adds MANAGE ACL for the current user to all secret scopes, ensuring
		// the migrated state and config agree on .permissions entries.
		bundle.ApplyContext(ctx, b, resourcemutator.SecretScopeFixups(engine.EngineDirect))
		if logdiag.HasError(ctx) {
			return root.ErrAlreadyPrinted
		}

		adapters, err := dresources.InitAll(nil)
		if err != nil {
			return err
		}

		if err := stateDB.UpgradeToWrite(); err != nil {
			return fmt.Errorf("upgrading state for apply: %w", err)
		}

		if err := migrate.BuildStateFromTF(ctx, &b.Config, adapters, &stateDB, tfAttrs, terraformResources); err != nil {
			return err
		}

		if _, err := stateDB.Finalize(ctx); err != nil {
			logdiag.LogError(ctx, err)
		}
		if logdiag.HasError(ctx) {
			logdiag.LogError(ctx, errors.New("migration failed; ensure you have done full deploy before the migration"))
			return root.ErrAlreadyPrinted
		}

		if err := os.Rename(tempStatePath, localPath); err != nil {
			return fmt.Errorf("renaming %s to %s: %w", tempStatePath, localPath, err)
		}
		tempStatePathAutoRemove = false

		localTerraformBackupPath := localTerraformPath + backupSuffix

		err = os.Rename(localTerraformPath, localTerraformBackupPath)
		if err != nil {
			// not fatal, since we've increased serial
			logdiag.LogError(ctx, err)
		}

		cmdio.LogString(ctx, fmt.Sprintf(`Success! Migrated %d resources to direct engine state file: %s

Validate the migration by running "databricks bundle plan%s", there should be no actions planned.

The state file is not synchronized to the workspace yet. To do that and finalize the migration, run "bundle deploy%s".

To undo the migration, remove %s and rename %s to %s
`, len(state), localPath, extraArgsStr, extraArgsStr, localPath, localTerraformBackupPath, localTerraformPath))
		return nil
	}

	return cmd
}

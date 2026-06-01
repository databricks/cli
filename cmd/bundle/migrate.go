package bundle

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config/engine"
	"github.com/databricks/cli/bundle/config/mutator/resourcemutator"
	"github.com/databricks/cli/bundle/deploy/terraform"
	"github.com/databricks/cli/bundle/direct/dresources"
	"github.com/databricks/cli/bundle/direct/dstate"
	"github.com/databricks/cli/bundle/migrate"
	"github.com/databricks/cli/cmd/bundle/utils"
	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/env"
	"github.com/databricks/cli/libs/logdiag"
	"github.com/spf13/cobra"
)

func newMigrateCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "migrate",
		Short: "Migrate from Terraform to Direct deployment engine (no API calls)",
		Long: `Creates a Direct deployment state file from the local config and Terraform state,
without making API calls. Cross-resource references are resolved from TF state.`,
		Args: root.NoArgs,
	}

	// --noplancheck is kept for compatibility but has no effect: this command reads
	// only from the local TF state file and never invokes the Terraform engine.
	cmd.Flags().Bool("noplancheck", false, "No-op (kept for compatibility).")

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		// Clear engine env var: we read TF state and produce a direct state.
		cmd.SetContext(env.Set(cmd.Context(), engine.EnvVar, ""))

		opts := utils.ProcessOptions{
			AlwaysPull:   true,
			FastValidate: true,
			Build:        true,
			PostInitFunc: func(_ context.Context, b *bundle.Bundle) error {
				if b.Config.Bundle.Engine == engine.EngineTerraform {
					return fmt.Errorf("bundle.engine is set to %q; migration requires \"engine: direct\" or no engine setting", engine.EngineTerraform)
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
			cmdio.LogString(ctx, `Error: no existing state found. To start fresh with direct engine, set "engine: direct".`)
			return root.ErrAlreadyPrinted
		}
		if stateDesc.Engine.IsDirect() {
			return fmt.Errorf("already using direct engine: %s", stateDesc.String())
		}

		_, localTerraformPath := b.StateFilenameTerraform(ctx)
		if _, err := os.Stat(localTerraformPath); err != nil {
			return fmt.Errorf("reading %s: %w", localTerraformPath, err)
		}

		// Parse TF state: IDs (for state entries) and full attributes (for ref resolution).
		tfResourceIDs, err := terraform.ParseResourcesState(ctx, b)
		if err != nil {
			return fmt.Errorf("failed to parse terraform state: %w", err)
		}
		for key, entry := range tfResourceIDs {
			if entry.ID == "" {
				return fmt.Errorf("missing ID for %s in terraform state", key)
			}
		}

		cacheDir, err := terraform.Dir(ctx, b)
		if err != nil {
			return err
		}
		tfStateFilename, _ := b.StateFilenameTerraform(ctx)
		tfStateFullPath := filepath.Join(cacheDir, tfStateFilename)

		tfAttrs, err := migrate.ParseTFStateAttrs(tfStateFullPath)
		if err != nil {
			return fmt.Errorf("failed to read terraform state attributes: %w", err)
		}

		_, localPath := b.StateFilenameDirect(ctx)
		tempPath := localPath + ".temp-migration"

		if _, err := os.Stat(tempPath); err == nil {
			return fmt.Errorf("temporary state file %s already exists, another migration is in progress or was interrupted. In the latter case, delete the file", tempPath)
		}
		if _, err := os.Stat(localPath); err == nil {
			return fmt.Errorf("state file %s already exists", localPath)
		}

		// Apply SecretScopeFixups so the config matches what the direct engine expects.
		// This adds MANAGE ACL for the current user to all secret scopes, ensuring
		// the migrated state and config agree on .permissions entries.
		bundle.ApplyContext(ctx, b, resourcemutator.SecretScopeFixups(engine.EngineDirect))
		if logdiag.HasError(ctx) {
			return root.ErrAlreadyPrinted
		}

		// Build initial state with IDs and optional ETags.
		etags := map[string]string{}
		state := make(map[string]dstate.ResourceEntry)
		for key, resourceEntry := range tfResourceIDs {
			state[key] = dstate.ResourceEntry{
				ID:    resourceEntry.ID,
				State: json.RawMessage("{}"),
			}
			if resourceEntry.ETag != "" {
				etags[key] = resourceEntry.ETag
			}
		}

		migratedDB := dstate.NewDatabase(stateDesc.Lineage, stateDesc.Serial+1)
		migratedDB.State = state

		var stateDB dstate.DeploymentState
		stateDB.OpenWithData(tempPath, migratedDB)

		removeTempPath := true
		defer func() {
			if removeTempPath {
				_ = os.Remove(tempPath)
			}
		}()

		// Initialize adapters.
		adapters, err := dresources.InitAll(b.WorkspaceClient(ctx))
		if err != nil {
			return err
		}

		if err := stateDB.UpgradeToWrite(); err != nil {
			return fmt.Errorf("upgrading state for write: %w", err)
		}

		// Process each resource: prepare state, resolve refs from TF state, save.
		if err := migrate.BuildStateFromTF(ctx, &b.Config, adapters, &stateDB, tfAttrs, tfResourceIDs, etags); err != nil {
			return err
		}

		if _, err := stateDB.Finalize(ctx); err != nil {
			return fmt.Errorf("finalizing state: %w", err)
		}
		if logdiag.HasError(ctx) {
			return errors.New("migration encountered errors")
		}

		if err := os.Rename(tempPath, localPath); err != nil {
			return fmt.Errorf("renaming %s to %s: %w", tempPath, localPath, err)
		}
		removeTempPath = false

		localTerraformBackupPath := localTerraformPath + ".backup"
		err = os.Rename(localTerraformPath, localTerraformBackupPath)
		if err != nil {
			// Not fatal since we've already incremented the serial.
			logdiag.LogError(ctx, err)
		}

		extraArgsStr := ""
		if flag := cmd.Flag("target"); flag != nil && flag.Changed {
			extraArgsStr = " -t " + flag.Value.String()
		}

		cmdio.LogString(ctx, fmt.Sprintf(`Success! Migrated %d resources to direct engine state file: %s

Validate the migration by running "databricks bundle plan%s", there should be no actions planned.

The state file is not synchronized to the workspace yet. To finalize the migration, run "bundle deploy%s".

To undo the migration, remove %s and rename %s to %s
`, len(state), localPath, extraArgsStr, extraArgsStr, localPath, localTerraformBackupPath, localTerraformPath))
		return nil
	}

	return cmd
}

package bundle

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"strings"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/engine"
	"github.com/databricks/cli/bundle/config/mutator/resourcemutator"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/bundle/deploy/terraform"
	"github.com/databricks/cli/bundle/direct"
	"github.com/databricks/cli/bundle/direct/dresources"
	"github.com/databricks/cli/bundle/direct/dstate"
	"github.com/databricks/cli/bundle/migrate"
	"github.com/databricks/cli/cmd/bundle/utils"
	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/env"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/cli/libs/logdiag"
	"github.com/databricks/cli/libs/structs/structaccess"
	"github.com/databricks/cli/libs/structs/structpath"
	"github.com/databricks/cli/libs/structs/structvar"
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
		if err := buildStateFromTF(ctx, b, adapters, &stateDB, tfAttrs, tfResourceIDs, etags); err != nil {
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

// buildStateFromTF iterates over bundle resources, resolves cross-resource
// references using TF state attributes, and writes each resource's state entry.
func buildStateFromTF(
	ctx context.Context,
	b *bundle.Bundle,
	adapters map[string]*dresources.Adapter,
	stateDB *dstate.DeploymentState,
	tfAttrs migrate.TFStateAttrs,
	tfIDs terraform.ExportedResourcesMap,
	etags map[string]string,
) error {
	configRoot := &b.Config

	// Collect all resource nodes (same patterns as makePlan).
	var nodes []string
	patterns := []dyn.Pattern{
		dyn.NewPattern(dyn.Key("resources"), dyn.AnyKey(), dyn.AnyKey()),
		dyn.NewPattern(dyn.Key("resources"), dyn.AnyKey(), dyn.AnyKey(), dyn.Key("permissions")),
		dyn.NewPattern(dyn.Key("resources"), dyn.AnyKey(), dyn.AnyKey(), dyn.Key("grants")),
	}
	for _, pat := range patterns {
		_, err := dyn.MapByPattern(
			configRoot.Value(),
			pat,
			func(p dyn.Path, v dyn.Value) (dyn.Value, error) {
				nodes = append(nodes, p.String())
				return dyn.InvalidValue, nil
			},
		)
		if err != nil {
			return err
		}
	}

	for _, node := range nodes {
		idEntry, ok := tfIDs[node]
		if !ok {
			// Resource is in config but not in TF state (new resource); skip.
			continue
		}

		group := config.GetResourceTypeFromKey(node)
		if group == "" {
			return fmt.Errorf("cannot determine resource type for %q", node)
		}

		adapter, ok := adapters[group]
		if !ok {
			log.Warnf(ctx, "unsupported resource type %q for %s, skipping", group, node)
			continue
		}

		inputConfig, err := configRoot.GetResourceConfig(node)
		if err != nil {
			return fmt.Errorf("%s: getting config: %w", node, err)
		}

		baseRefs := map[string]string{}

		switch {
		case strings.HasSuffix(node, ".permissions"):
			var sv *structvar.StructVar
			if strings.HasPrefix(node, "resources.secret_scopes.") {
				typedConfig, ok := inputConfig.(*[]resources.SecretScopePermission)
				if !ok {
					return fmt.Errorf("%s: expected *[]resources.SecretScopePermission, got %T", node, inputConfig)
				}
				sv, err = dresources.PrepareSecretScopeAclsInputConfig(*typedConfig, node)
				if err != nil {
					return fmt.Errorf("%s: preparing secret scope ACLs config: %w", node, err)
				}
			} else {
				sv, err = dresources.PreparePermissionsInputConfig(inputConfig, node)
				if err != nil {
					return fmt.Errorf("%s: preparing permissions config: %w", node, err)
				}
			}
			inputConfig = sv.Value
			baseRefs = sv.Refs

		case strings.HasSuffix(node, ".grants"):
			sv, err := dresources.PrepareGrantsInputConfig(inputConfig, node)
			if err != nil {
				return fmt.Errorf("%s: preparing grants config: %w", node, err)
			}
			inputConfig = sv.Value
			baseRefs = sv.Refs
		}

		newStateValue, err := adapter.PrepareState(inputConfig)
		if err != nil {
			return fmt.Errorf("%s: PrepareState: %w", node, err)
		}

		refs, err := direct.ExtractReferences(configRoot.Value(), node)
		if err != nil {
			return fmt.Errorf("%s: extracting references: %w", node, err)
		}
		maps.Copy(refs, baseRefs)

		sv := structvar.NewStructVar(newStateValue, refs)

		// Resolve each reference using TF state.
		// We need to extract the resource name for Method A (looking up in the source resource's TF state).
		parts := strings.SplitN(node, ".", 4)
		// node format: "resources.<group>.<name>" or "resources.<group>.<name>.permissions"
		var srcGroup, srcName string
		if len(parts) >= 3 {
			srcGroup = parts[1]
			srcName = parts[2]
		}

		// Collect all field paths that need resolution (avoid modifying map during iteration).
		type refEntry struct {
			fieldPathStr string
			refTemplate  string
		}
		var pendingRefs []refEntry
		for fieldPathStr, refTemplate := range sv.Refs {
			pendingRefs = append(pendingRefs, refEntry{fieldPathStr, refTemplate})
		}

		for _, pending := range pendingRefs {
			fieldPath, err := structpath.ParsePath(pending.fieldPathStr)
			if err != nil {
				return fmt.Errorf("%s: parsing field path %q: %w", node, pending.fieldPathStr, err)
			}

			// ResolveFieldRef returns the fully resolved value for this field,
			// using either Method A (TF state lookup) or Method B (template evaluation).
			value, err := migrate.ResolveFieldRef(ctx, tfAttrs, srcGroup, srcName, fieldPath, pending.refTemplate)
			if err != nil {
				return fmt.Errorf("%s: cannot resolve field %q (template %q): %w", node, pending.fieldPathStr, pending.refTemplate, err)
			}

			// Set the resolved value directly and remove the ref entry.
			if err := structaccess.Set(sv.Value, fieldPath, value); err != nil {
				return fmt.Errorf("%s: cannot set resolved value for field %q: %w", node, pending.fieldPathStr, err)
			}
			delete(sv.Refs, pending.fieldPathStr)
		}

		if len(sv.Refs) > 0 {
			return fmt.Errorf("%s: unresolved references: %v", node, sv.Refs)
		}

		// Handle etag for dashboards.
		if etag := etags[node]; etag != "" {
			if err := structaccess.Set(sv.Value, structpath.NewStringKey(nil, "etag"), etag); err != nil {
				return fmt.Errorf("%s: cannot set etag: %w", node, err)
			}
		}

		if err := stateDB.SaveState(node, idEntry.ID, sv.Value, nil); err != nil {
			return fmt.Errorf("%s: SaveState: %w", node, err)
		}
	}

	return nil
}

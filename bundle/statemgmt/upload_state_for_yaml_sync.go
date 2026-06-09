package statemgmt

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/engine"
	"github.com/databricks/cli/bundle/config/mutator/resourcemutator"
	"github.com/databricks/cli/bundle/deploy"
	"github.com/databricks/cli/bundle/deploy/terraform"
	"github.com/databricks/cli/bundle/deployplan"
	"github.com/databricks/cli/bundle/direct"
	"github.com/databricks/cli/bundle/direct/dstate"
	"github.com/databricks/cli/bundle/env"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/dyn/dynvar"
	"github.com/databricks/cli/libs/filer"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/cli/libs/logdiag"
	"github.com/databricks/cli/libs/structs/structaccess"
	"github.com/databricks/cli/libs/structs/structpath"
)

type uploadStateForYamlSync struct {
	engine engine.EngineType
}

// UploadStateForYamlSync converts the state to the direct format for YAML sync and uploads it to the Workspace.
// This is simplified version of the `bundle migrate` command.
// State file is saved in the same format as the direct engine but to the different path to avoid any side effects.
// This is temporary solution that needs to be removed once all bundles in DABs in the Workspace are migrated to the direct engine.
func UploadStateForYamlSync(targetEngine engine.EngineType) bundle.Mutator {
	return &uploadStateForYamlSync{engine: targetEngine}
}

func (m *uploadStateForYamlSync) Name() string {
	return "statemgmt.UploadStateForYamlSync"
}

func (m *uploadStateForYamlSync) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	if m.engine.IsDirect() {
		return nil
	}

	_, enabled := env.ExperimentalYamlSync(ctx)
	if !enabled {
		return nil
	}

	// convertState reuses direct-engine code (SecretScopeFixups, CalculatePlan, Apply)
	// that reports failures via logdiag on the context. This mutator is best-effort and
	// must not fail a deploy that already succeeded, so collect those diagnostics in an
	// isolated scope and downgrade them to warnings.
	ctx = logdiag.IsolatedContext(ctx)
	logdiag.SetCollect(ctx, true)
	defer func() {
		for _, d := range logdiag.FlushCollected(ctx) {
			msg := d.Summary
			if d.Detail != "" {
				msg += ": " + d.Detail
			}
			log.Warnf(ctx, "Config snapshot: %s", msg)
		}
	}()

	_, snapshotPath := b.StateFilenameConfigSnapshot(ctx)

	created, err := m.convertState(ctx, b, snapshotPath)
	if err != nil {
		log.Warnf(ctx, "Failed to create config snapshot: %v", err)
		return nil
	}
	if !created {
		return nil
	}

	err = uploadState(ctx, b)
	if err != nil {
		log.Warnf(ctx, "Failed to upload config snapshot: %v", err)
		return nil
	}

	log.Infof(ctx, "Config snapshot created at %s", snapshotPath)
	return nil
}

func uploadState(ctx context.Context, b *bundle.Bundle) error {
	f, err := deploy.StateFiler(ctx, b)
	if err != nil {
		return fmt.Errorf("failed to get state filer: %w", err)
	}

	remotePath, localPath := b.StateFilenameConfigSnapshot(ctx)
	local, err := os.Open(localPath)
	if errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("config snapshot file does not exist at %s", localPath)
	}
	if err != nil {
		return fmt.Errorf("failed to open config snapshot for upload: %w", err)
	}
	defer local.Close()

	err = f.Write(ctx, remotePath, local, filer.CreateParentDirectories, filer.OverwriteIfExists)
	if err != nil {
		return fmt.Errorf("failed to upload config snapshot to workspace: %w", err)
	}

	return nil
}

func (m *uploadStateForYamlSync) convertState(ctx context.Context, b *bundle.Bundle, snapshotPath string) (bool, error) {
	terraformResources, err := terraform.ParseResourcesState(ctx, b)
	if err != nil {
		return false, fmt.Errorf("failed to parse terraform state: %w", err)
	}

	// ParseResourcesState returns nil when the terraform state file doesn't exist
	// (e.g. first deploy with no resources).
	if terraformResources == nil {
		return false, nil
	}

	_, localTerraformPath := b.StateFilenameTerraform(ctx)
	data, err := os.ReadFile(localTerraformPath)
	if err != nil {
		return false, fmt.Errorf("failed to read terraform state: %w", err)
	}

	state := make(map[string]dstate.ResourceEntry)
	etags := map[string]string{}

	for key, resourceEntry := range terraformResources {
		state[key] = dstate.ResourceEntry{
			ID:    resourceEntry.ID,
			State: json.RawMessage("{}"),
		}
		if resourceEntry.ETag != "" {
			etags[key] = resourceEntry.ETag
		}
	}

	var tfState struct {
		Lineage string `json:"lineage"`
		Serial  int    `json:"serial"`
	}
	if err := json.Unmarshal(data, &tfState); err != nil {
		return false, err
	}

	migratedDB := dstate.NewDatabase(tfState.Lineage, tfState.Serial+1)
	migratedDB.State = state

	deploymentBundle := &direct.DeploymentBundle{}
	deploymentBundle.StateDB.OpenWithData(snapshotPath, migratedDB)

	// Apply SecretScopeFixups so the config matches what the direct engine expects.
	// This adds MANAGE ACL for the current user to all secret scopes, ensuring
	// the migrated state and config agree on .permissions entries.
	bundle.ApplyContext(ctx, b, resourcemutator.SecretScopeFixups(engine.EngineDirect))
	if logdiag.HasError(ctx) {
		return false, errors.New("failed to apply secret scope fixups")
	}

	// Get the dynamic value from b.Config and reverse the interpolation.
	// b.Config has been modified by terraform.Interpolate which converts bundle-style
	// references (${resources.pipelines.x.id}) to terraform-style (${databricks_pipeline.x.id}).
	interpolatedRoot := b.Config.Value()
	uninterpolatedRoot, err := reverseInterpolate(interpolatedRoot)
	if err != nil {
		return false, fmt.Errorf("failed to reverse interpolation: %w", err)
	}

	var uninterpolatedConfig config.Root
	err = uninterpolatedConfig.Mutate(func(_ dyn.Value) (dyn.Value, error) {
		return uninterpolatedRoot, nil
	})
	if err != nil {
		return false, fmt.Errorf("failed to create uninterpolated config: %w", err)
	}

	plan, err := deploymentBundle.CalculatePlan(ctx, b.WorkspaceClient(ctx), &uninterpolatedConfig)
	if err != nil {
		return false, err
	}

	for _, entry := range plan.Plan {
		entry.Action = deployplan.Update
	}

	for key := range plan.Plan {
		etag := etags[key]
		if etag == "" {
			continue
		}
		sv, ok := deploymentBundle.StateCache.Load(key)
		if !ok {
			continue
		}
		err := structaccess.Set(sv.Value, structpath.NewStringKey(nil, "etag"), etag)
		if err != nil {
			log.Warnf(ctx, "Failed to set etag on %q: %v", key, err)
		}
	}

	if err := deploymentBundle.StateDB.UpgradeToWrite(); err != nil {
		return false, fmt.Errorf("upgrading state for apply: %w", err)
	}

	deploymentBundle.Apply(ctx, b.WorkspaceClient(ctx), plan, direct.MigrateMode(true))
	if _, err := deploymentBundle.StateDB.Finalize(ctx); err != nil {
		return false, err
	}

	// Apply reports failures via logdiag instead of returning an error. Don't
	// upload a snapshot that is missing entries for the failed resources.
	if logdiag.HasError(ctx) {
		return false, errors.New("state conversion failed")
	}

	return true, nil
}

// reverseInterpolate reverses the terraform.Interpolate transformation.
// It converts terraform-style resource references back to bundle-style.
// Example: ${databricks_pipeline.my_etl.id} → ${resources.pipelines.my_etl.id}
func reverseInterpolate(root dyn.Value) (dyn.Value, error) {
	return dynvar.Resolve(root, func(path dyn.Path) (dyn.Value, error) {
		// Need at least 2 components: resource_type.resource_name
		if len(path) < 2 {
			return dyn.InvalidValue, dynvar.ErrSkipResolution
		}

		resourceType := path[0].Key()
		isAlreadyBundleFormat := resourceType == "resources"
		if isAlreadyBundleFormat {
			return dyn.InvalidValue, dynvar.ErrSkipResolution
		}

		bundleGroup, ok := terraform.TerraformToGroupName[resourceType]
		if !ok {
			return dyn.InvalidValue, dynvar.ErrSkipResolution
		}

		// Reconstruct path in bundle format:
		// databricks_pipeline.my_pipeline.id → resources.pipelines.my_pipeline.id
		bundlePath := dyn.NewPath(dyn.Key("resources"), dyn.Key(bundleGroup)).Append(path[1:]...)
		return dyn.V(fmt.Sprintf("${%s}", bundlePath.String())), nil
	})
}

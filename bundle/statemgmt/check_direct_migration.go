package statemgmt

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/engine"
	"github.com/databricks/cli/bundle/config/mutator/resourcemutator"
	"github.com/databricks/cli/bundle/direct/dresources"
	"github.com/databricks/cli/bundle/direct/dstate"
	"github.com/databricks/cli/bundle/metrics"
	"github.com/databricks/cli/bundle/migrate"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/cli/libs/logdiag"
)

// warnPrefix labels warnings emitted by the post-deploy dry-run so they are not
// confused with warnings from the user-invoked `bundle migrate` command.
const warnPrefix = "post-deploy dry-run migration to direct: "

// CheckDirectMigration performs a dry-run migration of the just-deployed terraform
// state to the direct engine and records the outcome in deploy telemetry.
//
// The converted state is written to a temporary file that is deleted before
// returning, so nothing is persisted locally or uploaded to the workspace; its
// only purpose is to measure, across the fleet, how many terraform deploys could
// migrate to the direct engine cleanly. Any error is surfaced to the user as a
// warning so it never fails a deploy that already succeeded.
func CheckDirectMigration(ctx context.Context, b *bundle.Bundle) {
	hasWarnings, err := dryRunMigrate(ctx, b)
	b.Metrics.SetBoolValue(metrics.DirectDryMigrateSuccess, err == nil)
	b.Metrics.SetBoolValue(metrics.DirectDryMigrateWarnings, hasWarnings)
	if err != nil {
		log.Warnf(ctx, "%s%v", warnPrefix, err)
	}
}

// dryRunMigrate converts the local terraform state to the direct engine state,
// returning whether any warnings were emitted. It mirrors the `bundle migrate`
// command but writes the result to a throwaway temp file that is deleted before
// returning, and never uploads anything.
func dryRunMigrate(ctx context.Context, b *bundle.Bundle) (bool, error) {
	_, localTerraformPath := b.StateFilenameTerraform(ctx)
	tfState, err := migrate.ParseTFStateFull(ctx, localTerraformPath)
	if err != nil {
		return false, fmt.Errorf("failed to parse terraform state: %w", err)
	}

	// ParseTFStateFull returns nil when the terraform state file doesn't exist
	// (e.g. first deploy with no resources); nothing to migrate, trivially OK.
	if tfState == nil {
		return false, nil
	}

	// The converted state is a throwaway: write it to a temp dir that is removed
	// (along with the WAL the state DB creates) before returning, so the dry run
	// leaves nothing behind on disk.
	tempDir, err := os.MkdirTemp("", "databricks-direct-migration-")
	if err != nil {
		return false, fmt.Errorf("failed to create temp dir: %w", err)
	}
	defer os.RemoveAll(tempDir)
	tempStatePath := filepath.Join(tempDir, "resources.json")

	// SecretScopeFixups and the direct-engine state builder report failures via
	// logdiag. Run them in an isolated context so a dry-run failure never affects
	// the deploy's own diagnostics or exit code.
	ctx = logdiag.IsolatedContext(ctx)

	state := make(map[string]dstate.ResourceEntry)
	for key, id := range tfState.IDs {
		state[key] = dstate.ResourceEntry{
			ID:    id,
			State: json.RawMessage("{}"),
		}
	}

	migratedDB := dstate.NewDatabase(tfState.Lineage, tfState.Serial+1)
	migratedDB.State = state

	var stateDB dstate.DeploymentState
	stateDB.OpenWithData(tempStatePath, migratedDB)

	// Apply SecretScopeFixups so the config matches what the direct engine expects.
	// This adds MANAGE ACL for the current user to all secret scopes, ensuring
	// the migrated state and config agree on .permissions entries.
	bundle.ApplyContext(ctx, b, resourcemutator.SecretScopeFixups(engine.EngineDirect))
	if logdiag.HasError(ctx) {
		return false, errors.New("failed to apply secret scope fixups")
	}

	// b.Config has been modified by terraform.Interpolate which converts bundle-style
	// references (${resources.pipelines.x.id}) to terraform-style (${databricks_pipeline.x.id}).
	// BuildStateFromTF expects ${resources.*} references, so reverse the interpolation first.
	uninterpolatedRoot, err := reverseInterpolate(b.Config.Value())
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

	adapters, err := dresources.InitAll(nil)
	if err != nil {
		return false, err
	}

	if err := stateDB.UpgradeToWrite(); err != nil {
		return false, fmt.Errorf("upgrading state for apply: %w", err)
	}

	// warnPrefix labels the conversion's warnings as coming from the background dry run.
	hasWarnings, err := migrate.BuildStateFromTF(ctx, &uninterpolatedConfig, adapters, &stateDB, tfState.Attrs, tfState.IDs, warnPrefix)
	if err != nil {
		return hasWarnings, err
	}

	if _, err := stateDB.Finalize(ctx); err != nil {
		return hasWarnings, err
	}

	// BuildStateFromTF reports some failures via logdiag instead of returning an error.
	if logdiag.HasError(ctx) {
		return hasWarnings, errors.New("state conversion failed")
	}

	return hasWarnings, nil
}

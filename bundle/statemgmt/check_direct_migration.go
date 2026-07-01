package statemgmt

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/engine"
	"github.com/databricks/cli/bundle/config/mutator/resourcemutator"
	"github.com/databricks/cli/bundle/direct/dresources"
	"github.com/databricks/cli/bundle/direct/dstate"
	"github.com/databricks/cli/bundle/migrate"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/cli/libs/logdiag"
	"github.com/databricks/cli/libs/telemetry"
	"github.com/databricks/cli/libs/telemetry/protos"
)

// CheckDirectMigration performs an in-memory dry-run migration of the just-deployed
// terraform state to the direct engine and records the outcome in deploy telemetry.
//
// It writes nothing to local disk or the workspace; its only purpose is to measure,
// across the fleet, how many terraform deploys could migrate to the direct engine
// cleanly. Any error is surfaced to the user as a warning so it never fails a deploy
// that already succeeded; warnings raised during the migration are downgraded to
// info level and reported only via telemetry.
func CheckDirectMigration(ctx context.Context, b *bundle.Bundle) {
	result := &protos.BundleDirectMigration{}
	b.Metrics.TerraformToDirectMigration = result

	hasWarnings, err := migrateInMemory(ctx, b)
	if err != nil {
		result.ErrorMessage = telemetry.ScrubErrorMessage(err.Error())
		log.Warnf(ctx, "Dry-run migration to direct engine failed: %v", err)
		return
	}

	result.Success = true
	result.HasWarnings = hasWarnings
}

// migrateInMemory converts the local terraform state to the direct engine state
// entirely in memory, returning whether any warnings were emitted. It mirrors the
// `bundle migrate` command but never persists or uploads the resulting state.
func migrateInMemory(ctx context.Context, b *bundle.Bundle) (bool, error) {
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
	stateDB.OpenInMemoryForWrite(migratedDB)

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

	// downgradeWarnings=true: warnings are reported via telemetry, not surfaced to
	// the user, since this is a background dry run on a deploy that already succeeded.
	hasWarnings, err := migrate.BuildStateFromTF(ctx, &uninterpolatedConfig, adapters, &stateDB, tfState.Attrs, tfState.IDs, true)
	if err != nil {
		return hasWarnings, err
	}

	// BuildStateFromTF reports some failures via logdiag instead of returning an error.
	if logdiag.HasError(ctx) {
		return hasWarnings, errors.New("state conversion failed")
	}

	return hasWarnings, nil
}

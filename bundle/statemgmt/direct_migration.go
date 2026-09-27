package statemgmt

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/engine"
	"github.com/databricks/cli/bundle/config/mutator/resourcemutator"
	"github.com/databricks/cli/bundle/deployplan"
	"github.com/databricks/cli/bundle/direct"
	"github.com/databricks/cli/bundle/direct/dresources"
	"github.com/databricks/cli/bundle/direct/dstate"
	"github.com/databricks/cli/bundle/metrics"
	"github.com/databricks/cli/bundle/migrate"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/cli/libs/logdiag"
)

// warnPrefix labels warnings emitted while converting terraform state to the direct
// engine, distinguishing them from the user-invoked "bundle deployment migrate".
const warnPrefix = "migration to direct: "

// Migrate converts the bundle's terraform state to a direct-engine state and opens
// b.DeploymentBundle.StateDB with it in memory. Returns false when there is no terraform state
// to migrate (the caller then opens the direct state normally).
//
// Migrate is reversible: nothing is written to the local state file or the workspace, so a
// command that does not commit - plan, run, or a declined deploy - just drops the in-memory
// state and stays on terraform. Callers that apply changes make the migration durable by calling
// CommitMigration once the command is approved.
//
// Any failure (parsing, conversion, the plan check) is non-fatal: a warning is emitted, false is
// returned, and the caller proceeds on the terraform engine, which is still in place - so a failed
// migration never blocks a command that would succeed. An empty terraform state goes through the
// same path: the converter writes an empty base state file, so there is no special case here.
func Migrate(ctx context.Context, b *bundle.Bundle, requiredEngine engine.EngineSetting) (bool, error) {
	_, localTerraformPath := b.StateFilenameTerraform(ctx)
	tfState, err := migrate.ParseTFStateFull(ctx, localTerraformPath)
	if err != nil {
		b.Metrics.SetBoolValue(metrics.DirectMigrateError, true)
		log.Warnf(ctx, "could not parse terraform state for migration to the direct engine; deploying on terraform this time: %v", err)
		return false, nil
	}
	if tfState == nil {
		return false, nil
	}

	_, localDirectPath := b.StateFilenameDirect(ctx)

	tempStatePath, _, hasWarnings, cfg, err := convertTFStateToDirect(ctx, b, tfState)
	if tempStatePath != "" {
		// Always remove the temp state and its WAL. The converted state is loaded into memory
		// below and never renamed into place, so these are the only on-disk copies to clean up.
		defer func() {
			_ = os.Remove(tempStatePath)
			_ = os.Remove(tempStatePath + ".wal")
		}()
	}
	if hasWarnings {
		b.Metrics.SetBoolValue(metrics.DirectMigrateWarnings, true)
	}
	if err != nil {
		b.Metrics.SetBoolValue(metrics.DirectMigrateError, true)
		log.Warnf(ctx, "could not convert terraform state to the direct engine; deploying on terraform this time: %v", err)
		return false, nil
	}

	// Plan-check the converted state: validate it can be planned - falling back to terraform if
	// not - and record the recreate metric.
	plan, err := checkPlanOnTempState(ctx, b, tempStatePath, cfg)
	if err != nil {
		b.Metrics.SetBoolValue(metrics.DirectMigratePlanError, true)
		log.Warnf(ctx, "migration to the direct engine failed its plan check; deploying on terraform this time: %v", err)
		return false, nil
	}

	// Record when the migrated state's first plan would recreate a resource, but still migrate:
	// such a recreate is a genuine pending config change (the same one terraform would apply), and
	// the command's approval flow gates it behind --auto-approve exactly like any other recreate.
	if recreated := recreatedResources(plan); len(recreated) > 0 {
		b.Metrics.SetBoolValue(metrics.DirectMigrateRecreatePlanned, true)
		log.Infof(ctx, "migration to the direct engine will recreate %v; the command's approval gates this", recreated)
	}

	// Load the converted state into memory only (serial tf+1). It was just written by this CLI, so
	// it is at the current schema version and needs no migration. Nothing is persisted or pushed
	// here; CommitMigration does that once the command is approved.
	raw, err := os.ReadFile(tempStatePath)
	if err != nil {
		return false, fmt.Errorf("reading migrated state: %w", err)
	}
	var data dstate.Database
	if err := json.Unmarshal(raw, &data); err != nil {
		return false, fmt.Errorf("parsing migrated state: %w", err)
	}
	b.DeploymentBundle.StateDB.OpenWithData(localDirectPath, data)

	// Announce the migration only once the conversion and plan check succeeded, so a run that
	// falls back to terraform above says nothing misleading.
	cmdio.LogString(ctx, "Notice: migrating your bundle to direct deployment engine (https://github.com/databricks/cli/issues/6765).")
	return true, nil
}

// CleanupTerraformStateAfterMigration backs up the now-superseded terraform state, remote and
// local, best-effort. Called by CommitMigration once the migrated direct state is pushed, so a
// later run does not pick up the stale terraform state (its resources are gone) instead of the
// direct one. Destroy does not use this: it removes the local terraform state directly in
// destroyCore and files.Delete removes the remote one.
func CleanupTerraformStateAfterMigration(ctx context.Context, b *bundle.Bundle) {
	BackupRemoteTerraformState(ctx, b)
	_, localTerraformPath := b.StateFilenameTerraform(ctx)
	if err := os.Rename(localTerraformPath, localTerraformPath+".backup"); err != nil && !errors.Is(err, fs.ErrNotExist) {
		log.Warnf(ctx, "automatic migration to direct engine: could not back up local terraform state: %v", err)
	}
}

// CommitMigration makes a prepared (in-memory) migration durable - the point of no return. It
// persists the converted direct state locally (serial tf+1), pushes it to the workspace, retires
// the superseded terraform state (local rename + remote backup), and records which source
// triggered the migration. Deploy calls it once the deploy is approved, before applying its own
// changes (which advance the state to tf+2). If the state push fails the terraform state is left
// in place, so the migration is not committed and a retry can complete it. Destroy does not call
// this: it commits the migration as part of its teardown (files.Delete owns the remote terraform
// state, and a destroy that migrates only to tear down is not adoption worth recording).
func CommitMigration(ctx context.Context, b *bundle.Bundle, requiredEngine engine.EngineSetting) {
	count := len(b.DeploymentBundle.StateDB.ExportState(ctx))
	if err := b.DeploymentBundle.StateDB.Persist(); err != nil {
		logdiag.LogError(ctx, fmt.Errorf("persisting migrated direct state: %w", err))
		return
	}
	PushResourcesState(ctx, b, engine.EngineDirect)
	if logdiag.HasError(ctx) {
		return
	}
	CleanupTerraformStateAfterMigration(ctx, b)
	recordAutoMigrateSource(b, requiredEngine)

	suffix := "s"
	if count == 1 {
		suffix = ""
	}
	cmdio.LogString(ctx, fmt.Sprintf("Migrated %d resource%s to direct deployment engine.", count, suffix))
}

// recordAutoMigrateSource sets exactly one of the migrated-via-* telemetry keys, per how
// the direct engine was selected. ConfigType is set only when bundle.engine populated it,
// so it distinguishes a durable opt-in (via_config) from an env-var-only one (via_env);
// IsDefault covers the population that asked for nothing.
func recordAutoMigrateSource(b *bundle.Bundle, requiredEngine engine.EngineSetting) {
	switch {
	case requiredEngine.IsDefault:
		b.Metrics.SetBoolValue(metrics.DirectAutoMigrateViaDefault, true)
	case requiredEngine.ConfigType == engine.EngineDirect:
		b.Metrics.SetBoolValue(metrics.DirectAutoMigrateViaConfig, true)
	default:
		b.Metrics.SetBoolValue(metrics.DirectAutoMigrateViaEnv, true)
	}
}

// DryRunMigrationTelemetry converts the terraform state to the direct engine WITHOUT
// committing, purely to record direct_drymigrate_* telemetry for deploys that opted out
// of the direct engine (engine: terraform). It mirrors the actual migration's conversion
// (but never runs the plan check, and never touches any state) so the fleet-wide "could
// this bundle migrate?" signal is preserved. It is called after a terraform deploy, so
// mutating b.Config during the conversion is harmless, and it swallows failures because
// the deploy already succeeded. DirectDryMigrateSuccess reflects only whether the state
// conversion succeeded.
func DryRunMigrationTelemetry(ctx context.Context, b *bundle.Bundle) {
	_, localTerraformPath := b.StateFilenameTerraform(ctx)
	tfState, err := migrate.ParseTFStateFull(ctx, localTerraformPath)
	if err != nil {
		b.Metrics.SetBoolValue(metrics.DirectDryMigrateSuccess, false)
		return
	}
	if tfState == nil {
		return
	}
	// An empty terraform state has nothing to convert, so the dry run trivially succeeds.
	if len(tfState.IDs) == 0 && len(tfState.Attrs) == 0 {
		b.Metrics.SetBoolValue(metrics.DirectDryMigrateSuccess, true)
		b.Metrics.SetBoolValue(metrics.DirectDryMigrateWarnings, false)
		return
	}

	tempStatePath, _, hasWarnings, _, err := convertTFStateToDirect(ctx, b, tfState)
	if tempStatePath != "" {
		defer func() {
			_ = os.Remove(tempStatePath)
			_ = os.Remove(tempStatePath + ".wal")
		}()
	}
	b.Metrics.SetBoolValue(metrics.DirectDryMigrateSuccess, err == nil)
	b.Metrics.SetBoolValue(metrics.DirectDryMigrateWarnings, hasWarnings)
}

// checkPlanOnTempState opens the migrated state at tempStatePath in read mode,
// runs a full plan against it, and returns the plan (and a non-nil error if the
// plan fails). Individual planning errors are emitted as warnings with warnPrefix
// so they are visible without failing the deploy. The plan is run in an isolated
// context so its diagnostics do not affect the deploy's own error state. The
// returned plan lets the caller inspect the planned actions (e.g. reject a
// migration that would recreate a resource).
func checkPlanOnTempState(ctx context.Context, b *bundle.Bundle, tempStatePath string, cfg *config.Root) (*deployplan.Plan, error) {
	planCtx := logdiag.IsolatedContext(ctx)
	logdiag.SetCollect(planCtx, true)
	defer func() {
		for _, d := range logdiag.FlushCollected(planCtx) {
			msg := d.Summary
			if d.Detail != "" {
				msg += ": " + d.Detail
			}
			log.Warnf(ctx, "%s%s", warnPrefix, msg)
		}
	}()

	var planBundle direct.DeploymentBundle

	// This plan is not created with the deployment history feature enabled,
	// so we can safely pass false for withDeploymentHistory.
	if err := planBundle.StateDB.Open(planCtx, tempStatePath, false, false, dstate.WithDeploymentHistory(false), dstate.OpenDmsArgs{}); err != nil {
		return nil, fmt.Errorf("opening migrated state for plan check: %w", err)
	}

	return planBundle.CalculatePlan(planCtx, b.WorkspaceClient(ctx), cfg)
}

// recreatedResources returns the sorted keys of resources the plan would recreate
// (destroy + create). Used only to record the direct_migrate_recreate_planned metric:
// the migration proceeds regardless, and the deploy's approval flow gates the recreate
// behind --auto-approve just like any other recreate.
func recreatedResources(plan *deployplan.Plan) []string {
	var keys []string
	for key, entry := range plan.Plan {
		if entry.Action == deployplan.Recreate {
			keys = append(keys, key)
		}
	}
	slices.Sort(keys)
	return keys
}

// convertTFStateToDirect converts the given terraform state to the direct engine state,
// returning the path to the converted state file, the number of resources
// migrated, whether any warnings were emitted, and the bundle config with
// terraform interpolation reversed (needed by the caller to run a plan against
// the converted state). Callers must ensure tfState is non-nil; an empty state
// (no resource IDs or attrs) yields an empty base state file. The caller is
// responsible for deleting the temp state's parent directory when it is done with the file.
func convertTFStateToDirect(ctx context.Context, b *bundle.Bundle, tfState *migrate.TFState) (string, int, bool, *config.Root, error) {
	// Write the converted state to a sibling of the final resources.json
	// path so commitMigration's os.Rename stays within one filesystem
	// (os.TempDir() often lives on a different volume from the project;
	// cross-filesystem Rename fails with EXDEV). The state DB creates the
	// file and its .wal itself, so a deterministic sibling name is enough
	// — no CreateTemp placeholder needed.
	// UpgradeToWrite creates the parent directory itself, so no MkdirAll here.
	_, localDirectPath := b.StateFilenameDirect(ctx)
	tempStatePath := filepath.Join(filepath.Dir(localDirectPath), "resources.migrating.json")
	// Clean up any leftovers from a crashed previous run so UpgradeToWrite
	// (which opens the .wal with O_EXCL) succeeds.
	_ = os.Remove(tempStatePath)
	_ = os.Remove(tempStatePath + ".wal")
	resourceCount := len(tfState.IDs)

	// SecretScopeFixups and the direct-engine state builder report failures via
	// logdiag. Run them in an isolated + collecting context so their diagnostics
	// neither affect the deploy's exit code (isolated) nor render as user-facing
	// `Error:` lines (collected + re-logged as warnings below).
	ctx = logdiag.IsolatedContext(ctx)
	logdiag.SetCollect(ctx, true)
	defer func() {
		for _, d := range logdiag.FlushCollected(ctx) {
			msg := d.Summary
			if d.Detail != "" {
				msg += ": " + d.Detail
			}
			log.Warnf(ctx, "%s%s", warnPrefix, msg)
		}
	}()

	var stateDB dstate.DeploymentState
	stateDB.OpenWithData(tempStatePath, dstate.NewDatabase(tfState.Lineage, tfState.Serial+1))

	// An empty terraform state seeds and builds no WAL entries below, so Finalize would persist
	// no file. Write the base file now so the migration always yields one; the deferred commit
	// builds on it and a crash mid-apply stays recoverable. The header-only Finalize leaves it
	// intact.
	if len(tfState.IDs) == 0 && len(tfState.Attrs) == 0 {
		if err := stateDB.Persist(); err != nil {
			return tempStatePath, resourceCount, false, nil, fmt.Errorf("persisting empty migrated state: %w", err)
		}
	}

	// Apply SecretScopeFixups so the config matches what the direct engine expects.
	// This adds MANAGE ACL for the current user to all secret scopes, ensuring
	// the migrated state and config agree on .permissions entries.
	bundle.ApplyContext(ctx, b, resourcemutator.SecretScopeFixups(engine.EngineDirect))
	if logdiag.HasError(ctx) {
		return tempStatePath, resourceCount, false, nil, errors.New("failed to apply secret scope fixups")
	}

	// b.Config has been modified by terraform.Interpolate which converts bundle-style
	// references (${resources.pipelines.x.id}) to terraform-style (${databricks_pipeline.x.id}).
	// BuildStateFromTF expects ${resources.*} references, so reverse the interpolation first.
	uninterpolatedRoot, err := reverseInterpolate(b.Config.Value())
	if err != nil {
		return tempStatePath, resourceCount, false, nil, fmt.Errorf("failed to reverse interpolation: %w", err)
	}

	var uninterpolatedConfig config.Root
	err = uninterpolatedConfig.Mutate(func(_ dyn.Value) (dyn.Value, error) {
		return uninterpolatedRoot, nil
	})
	if err != nil {
		return tempStatePath, resourceCount, false, nil, fmt.Errorf("failed to create uninterpolated config: %w", err)
	}

	adapters, err := dresources.InitAll(nil)
	if err != nil {
		return tempStatePath, resourceCount, false, nil, err
	}

	if err := stateDB.UpgradeToWrite(); err != nil {
		return tempStatePath, resourceCount, false, nil, fmt.Errorf("upgrading state for apply: %w", err)
	}

	// Seed every terraform-state resource into the WAL so the migrated state persists
	// all deployed resources, including ones the current config no longer declares.
	// BuildStateFromTF overwrites the config-declared entries below with their full
	// state (the later WAL entry wins on replay); the rest keep this minimal entry,
	// which is enough for the first direct plan to delete them. Without this, a config
	// that dropped every resource would record no WAL entries at all, so Finalize would
	// persist no state file and the migration would fail with a missing resources.json.
	for key, id := range tfState.IDs {
		if err := stateDB.SaveState(ctx, key, id, json.RawMessage("{}"), nil); err != nil {
			return tempStatePath, resourceCount, false, nil, fmt.Errorf("seeding migrated state for %s: %w", key, err)
		}
	}

	// warnPrefix labels the conversion's warnings as coming from the background dry run.
	hasWarnings, err := migrate.BuildStateFromTF(ctx, &uninterpolatedConfig, adapters, &stateDB, tfState.Attrs, tfState.IDs, warnPrefix)
	if err != nil {
		return tempStatePath, resourceCount, hasWarnings, nil, err
	}

	if _, err := stateDB.Finalize(ctx); err != nil {
		return tempStatePath, resourceCount, hasWarnings, nil, err
	}

	// BuildStateFromTF reports some failures via logdiag instead of returning an error.
	if logdiag.HasError(ctx) {
		return tempStatePath, resourceCount, hasWarnings, nil, errors.New("state conversion failed")
	}

	return tempStatePath, resourceCount, hasWarnings, &uninterpolatedConfig, nil
}

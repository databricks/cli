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
	"github.com/databricks/cli/libs/env"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/cli/libs/logdiag"
)

// warnPrefix labels warnings emitted while converting terraform state to the direct
// engine, distinguishing them from the user-invoked "bundle deployment migrate".
const warnPrefix = "migration to direct: "

// migrationFailedHint is appended to the errors that abort an automatic terraform→direct
// migration. The terraform engine was removed in v1.20.0, so there is no engine left to fall
// back to: the command cannot proceed until the state migrates or the user downgrades.
const migrationFailedHint = `

The Terraform deployment engine was removed in Databricks CLI v1.20.0. Auto-migration of your bundle failed. Install Databricks CLI v1.19.x to deploy on Terraform, and please report this to dabs-feedback@databricks.com`

// Migrate converts the bundle's terraform state to a direct-engine state and opens
// b.DeploymentBundle.StateDB with it in memory. Returns false when there is no terraform state
// to migrate (the caller then opens the direct state normally).
//
// Migrate is reversible: nothing is written to the local state file or the workspace, so a
// command that does not commit - plan, run, or a declined deploy - just drops the in-memory
// state. Callers that apply changes make the migration durable by calling CommitMigration once
// the command is approved.
//
// The terraform engine was removed in v1.20.0, so any failure (parsing, conversion, the plan
// check) is fatal: there is no engine to fall back to. An empty terraform state is not a failure -
// the converter writes an empty base state file, so there is no special case here.
func Migrate(ctx context.Context, b *bundle.Bundle) (bool, error) {
	_, localTerraformPath := b.StateFilenameTerraform(ctx)
	tfState, err := migrate.ParseTFStateFull(ctx, localTerraformPath)
	if err != nil {
		b.Metrics.SetBoolValue(metrics.DirectMigrateError, true)
		return false, fmt.Errorf("parsing terraform state: %w%s", err, migrationFailedHint)
	}
	if tfState == nil {
		return false, nil
	}

	_, localDirectPath := b.StateFilenameDirect(ctx)

	tempStatePath, hasWarnings, cfg, err := convertTFStateToDirect(ctx, b, tfState)
	if tempStatePath != "" {
		// Always remove the temp state and its WAL. The converted state is loaded into memory
		// below and never renamed into place, so these are the only on-disk copies to clean up.
		defer dstate.RemoveFiles(tempStatePath)
	}
	if hasWarnings {
		b.Metrics.SetBoolValue(metrics.DirectMigrateWarnings, true)
	}
	if err != nil {
		b.Metrics.SetBoolValue(metrics.DirectMigrateError, true)
		return false, fmt.Errorf("converting terraform state to the direct engine: %w%s", err, migrationFailedHint)
	}

	// Plan-check the converted state: validate it can be planned before committing to it, and
	// record the recreate metric.
	plan, err := checkPlanOnTempState(ctx, b, tempStatePath, cfg)
	if err != nil {
		b.Metrics.SetBoolValue(metrics.DirectMigratePlanError, true)
		return false, fmt.Errorf("the migrated state failed its plan check: %w%s", err, migrationFailedHint)
	}

	// Record when the migrated state's first plan would recreate a resource, but still migrate:
	// such a recreate is a genuine pending config change (the same one terraform would apply), and
	// the command's approval flow gates it behind --auto-approve exactly like any other recreate.
	if recreated := recreatedResources(plan); len(recreated) > 0 {
		b.Metrics.SetBoolValue(metrics.DirectMigrateRecreatePlanned, true)
		log.Infof(ctx, "migration to the direct engine will recreate %v; the command's approval gates this", recreated)
	}

	// Load the converted state into memory only (at a serial above the terraform state's - see
	// convertTFStateToDirect). It was just written by this CLI, so it is at the current schema
	// version and needs no migration. Nothing is persisted or pushed here; CommitMigration does
	// that once the command is approved.
	raw, err := os.ReadFile(tempStatePath)
	if err != nil {
		return false, fmt.Errorf("reading migrated state: %w", err)
	}
	var data dstate.Database
	if err := json.Unmarshal(raw, &data); err != nil {
		return false, fmt.Errorf("parsing migrated state: %w", err)
	}
	b.DeploymentBundle.StateDB.OpenWithData(localDirectPath, data)

	// Announce the migration only once the conversion and plan check succeeded, so a run with no
	// terraform state to migrate says nothing misleading.
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
// persists the converted direct state locally (at a serial above the terraform state's), pushes it
// to the workspace, retires the superseded terraform state (local rename + remote backup), and
// records which source triggered the migration. Deploy calls it once the deploy is approved, before
// applying its own changes (which advance the state once more). If the state push fails the terraform state is left
// in place, so the migration is not committed and a retry can complete it. Destroy does not call
// this: it commits the migration as part of its teardown (files.Delete owns the remote terraform
// state, and a destroy that migrates only to tear down is not adoption worth recording).
func CommitMigration(ctx context.Context, b *bundle.Bundle) {
	count := len(b.DeploymentBundle.StateDB.ExportState(ctx))
	if err := b.DeploymentBundle.StateDB.Persist(); err != nil {
		logdiag.LogError(ctx, fmt.Errorf("persisting migrated direct state: %w", err))
		return
	}
	PushResourcesState(ctx, b)
	if logdiag.HasError(ctx) {
		return
	}
	CleanupTerraformStateAfterMigration(ctx, b)
	recordAutoMigrateSource(ctx, b)

	suffix := "s"
	if count == 1 {
		suffix = ""
	}
	cmdio.LogString(ctx, fmt.Sprintf("Migrated %d resource%s to direct deployment engine.", count, suffix))
}

// recordAutoMigrateSource sets exactly one of the migrated-via-* telemetry keys, per how
// the direct engine was selected: a durable opt-in in bundle.engine (via_config), an
// env-var-only one (via_env), or nothing at all (via_default). The setting was already
// validated, so a set value can only be "direct", and bundle.engine wins over the env var.
func recordAutoMigrateSource(ctx context.Context, b *bundle.Bundle) {
	switch {
	case b.Config.Bundle.Engine != engine.EngineNotSet:
		b.Metrics.SetBoolValue(metrics.DirectAutoMigrateViaConfig, true)
	case env.Get(ctx, engine.EnvVar) != "":
		b.Metrics.SetBoolValue(metrics.DirectAutoMigrateViaEnv, true)
	default:
		b.Metrics.SetBoolValue(metrics.DirectAutoMigrateViaDefault, true)
	}
}

// checkPlanOnTempState opens the migrated state at tempStatePath in read mode,
// runs a full plan against it, and returns the plan, or a non-nil error if the
// plan fails (the caller fails the migration on it). Diagnostics collected while planning
// are additionally logged as warnings with warnPrefix for visibility. The plan is run in an isolated
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

// convertTFStateToDirect converts the given terraform state to a direct-engine state file,
// returning the path to that file, whether any warnings were emitted, and the bundle config with
// terraform interpolation reversed (needed by the caller to run a plan against the converted
// state). Callers must ensure tfState is non-nil; an empty state (no resource IDs or attrs) yields
// an empty base state file. The caller reads the file into memory and is responsible for removing
// it (and its .wal) when done.
func convertTFStateToDirect(ctx context.Context, b *bundle.Bundle, tfState *migrate.TFState) (string, bool, *config.Root, error) {
	// Write the converted state to a deterministic sibling of the final resources.json path. The
	// caller (Migrate) reads it into memory with OpenWithData and then removes it; nothing renames
	// it into place. The state DB creates the file and its .wal itself, and UpgradeToWrite creates
	// the parent directory, so no CreateTemp or MkdirAll is needed here.
	_, localDirectPath := b.StateFilenameDirect(ctx)
	tempStatePath := filepath.Join(filepath.Dir(localDirectPath), "resources.migrating.json")
	// Clean up any leftovers from a crashed previous run so UpgradeToWrite
	// (which opens the .wal with O_EXCL) succeeds.
	dstate.RemoveFiles(tempStatePath)

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

	// The converted state starts one serial above the terraform state. A populated conversion then
	// replays a WAL below (SaveState + BuildStateFromTF), which advances the serial once more - so a
	// populated migration lands at tf+2 and an empty one (persisted directly below, no WAL entries)
	// stays at tf+1. Either way it outranks the terraform state, which is all engine selection needs.
	var stateDB dstate.DeploymentState
	stateDB.OpenWithData(tempStatePath, dstate.NewDatabase(tfState.Lineage, tfState.Serial+1))
	// On a failed conversion, close the WAL so the caller can remove the files (Windows refuses to
	// remove an open file). After the successful Finalize below this is a no-op.
	defer stateDB.Discard()

	// Apply SecretScopeFixups so the config matches what the direct engine expects.
	// This adds MANAGE ACL for the current user to all secret scopes, ensuring
	// the migrated state and config agree on .permissions entries.
	bundle.ApplyContext(ctx, b, resourcemutator.SecretScopeFixups())
	if logdiag.HasError(ctx) {
		return tempStatePath, false, nil, errors.New("failed to apply secret scope fixups")
	}

	// The config may use terraform-style references (${databricks_pipeline.x.id}).
	// BuildStateFromTF expects ${resources.*} references, so rewrite them first.
	uninterpolatedRoot, err := reverseInterpolate(b.Config.Value())
	if err != nil {
		return tempStatePath, false, nil, fmt.Errorf("failed to reverse interpolation: %w", err)
	}

	var uninterpolatedConfig config.Root
	err = uninterpolatedConfig.Mutate(func(_ dyn.Value) (dyn.Value, error) {
		return uninterpolatedRoot, nil
	})
	if err != nil {
		return tempStatePath, false, nil, fmt.Errorf("failed to create uninterpolated config: %w", err)
	}

	adapters, err := dresources.InitAll(nil)
	if err != nil {
		return tempStatePath, false, nil, err
	}

	if err := stateDB.UpgradeToWrite(); err != nil {
		return tempStatePath, false, nil, fmt.Errorf("upgrading state for apply: %w", err)
	}

	if err := migrate.SeedState(ctx, &stateDB, tfState); err != nil {
		return tempStatePath, false, nil, err
	}

	// warnPrefix labels the conversion's warnings as coming from the background dry run.
	hasWarnings, err := migrate.BuildStateFromTF(ctx, &uninterpolatedConfig, adapters, &stateDB, tfState.Attrs, tfState.IDs, warnPrefix)
	if err != nil {
		return tempStatePath, hasWarnings, nil, err
	}

	if _, err := stateDB.Finalize(ctx); err != nil {
		return tempStatePath, hasWarnings, nil, err
	}

	// BuildStateFromTF reports some failures via logdiag instead of returning an error.
	if logdiag.HasError(ctx) {
		return tempStatePath, hasWarnings, nil, errors.New("state conversion failed")
	}

	return tempStatePath, hasWarnings, &uninterpolatedConfig, nil
}

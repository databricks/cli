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
	"github.com/databricks/cli/bundle/deploy"
	"github.com/databricks/cli/bundle/deployplan"
	"github.com/databricks/cli/bundle/direct"
	"github.com/databricks/cli/bundle/direct/dresources"
	"github.com/databricks/cli/bundle/direct/dstate"
	"github.com/databricks/cli/bundle/metrics"
	"github.com/databricks/cli/bundle/migrate"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/filer"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/cli/libs/logdiag"
)

// warnPrefix labels warnings emitted while converting terraform state to the direct
// engine, distinguishing them from the user-invoked "bundle deployment migrate".
const warnPrefix = "migration to direct: "

// MigrateMode selects how MigrateTerraformState commits the converted state.
type MigrateMode int

const (
	// MigratePlan converts the state and loads it into memory only (for `bundle plan` and
	// `bundle run`): nothing is written or pushed and no plan check runs.
	MigratePlan MigrateMode = iota

	// MigrateDeferred converts the state, plan-checks it (falling back to terraform if the
	// check fails), writes the local direct state file, and sets b.MigrationDeferred. Used by
	// the commands that apply changes (deploy, destroy): the command commits the state itself
	// once it is approved (deployCore / destroyCore) and then calls FinalizeDeferredMigration
	// to clean up the terraform state; a declined command discards it (DiscardDeferredMigration)
	// and stays on terraform.
	MigrateDeferred
)

// MigrateTerraformState converts the bundle's terraform state to a direct-engine state
// and opens b.DeploymentBundle.StateDB with it. Returns false when there is no terraform
// state to migrate (the caller then opens the direct state normally).
//
// MigratePlan loads the converted state in memory and writes nothing. MigrateDeferred also
// plan-checks it, writes the local direct state file, and sets b.MigrationDeferred so the
// approved command commits it and calls FinalizeDeferredMigration; a declined command calls
// DiscardDeferredMigration. The workspace is never touched here - it is committed by the
// command's own state push after approval.
//
// Any failure (parsing, conversion, the empty-state sweep, the plan check) is non-fatal: a
// warning is emitted, false is returned, and the caller proceeds on the terraform engine,
// which is still in place - so a failed migration never blocks a command that would succeed.
func MigrateTerraformState(ctx context.Context, b *bundle.Bundle, requiredEngine engine.EngineSetting, mode MigrateMode) (bool, error) {
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

	// A terraform state with no managed resources carries nothing to migrate and cannot
	// produce a destructive plan, so commit it up front for deploy and destroy (there is
	// nothing to defer): sweep the empty terraform state aside so the direct engine is used
	// from now on. Plan just opens an empty direct database in memory. No resources.json is
	// written.
	if len(tfState.IDs) == 0 && len(tfState.Attrs) == 0 {
		if mode == MigrateDeferred {
			cmdio.LogString(ctx, "Removing empty terraform state; the direct engine will be used from now on...")
			if err := BackupTerraformState(ctx, b); err != nil {
				b.Metrics.SetBoolValue(metrics.DirectMigrateCommitError, true)
				log.Warnf(ctx, "could not remove empty terraform state for migration to the direct engine; deploying on terraform this time: %v", err)
				return false, nil
			}
			recordAutoMigrateSource(b, requiredEngine)
		}
		b.DeploymentBundle.StateDB.OpenWithData(localDirectPath, dstate.NewDatabase(tfState.Lineage, tfState.Serial+1))
		return true, nil
	}

	tempStatePath, resourceCount, hasWarnings, cfg, err := convertTFStateToDirect(ctx, b, tfState)
	if tempStatePath != "" {
		// Always remove the temp state and its WAL. A successful commit renames the temp
		// file into place first, so these become no-ops; on any failure they clean up the
		// leftovers so a later run's UpgradeToWrite (which opens the WAL with O_EXCL) does
		// not trip over them.
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

	// Plan-check the converted state for deploy and destroy (not plan): validate it can be
	// planned - falling back to terraform if not - and record the recreate metric.
	if mode == MigrateDeferred {
		plan, err := checkPlanOnTempState(ctx, b, tempStatePath, cfg)
		if err != nil {
			b.Metrics.SetBoolValue(metrics.DirectMigratePlanError, true)
			log.Warnf(ctx, "migration to the direct engine failed its plan check; deploying on terraform this time: %v", err)
			return false, nil
		}

		// Record when the migrated state's first plan would recreate a resource, but still
		// migrate: such a recreate is a genuine pending config change (the same one
		// terraform would apply), and the command's approval flow gates it behind
		// --auto-approve exactly like any other recreate. The metric lets us observe how
		// often a migration carries a recreate.
		if recreated := recreatedResources(plan); len(recreated) > 0 {
			b.Metrics.SetBoolValue(metrics.DirectMigrateRecreatePlanned, true)
			log.Infof(ctx, "migration to the direct engine will recreate %v; the command's approval gates this", recreated)
		}
	}

	if mode == MigratePlan {
		// Plan never persists the state, so load the converted state into memory only. It was
		// just written by this CLI, so it is at the current schema version and needs no migration.
		raw, err := os.ReadFile(tempStatePath)
		if err != nil {
			return false, fmt.Errorf("reading migrated state: %w", err)
		}
		var data dstate.Database
		if err := json.Unmarshal(raw, &data); err != nil {
			return false, fmt.Errorf("parsing migrated state: %w", err)
		}
		b.DeploymentBundle.StateDB.OpenWithData(localDirectPath, data)
		return true, nil
	}

	// MigrateDeferred: place the converted state in the local direct state file so the command
	// (deployCore / destroyCore) has a base to persist even when the deploy is a no-op - an
	// in-memory state is dropped by Finalize when there are no operations. The remote push and
	// terraform-state cleanup are deferred to the approved command (its own state push +
	// FinalizeDeferredMigration), so the workspace is untouched until then; a declined command
	// removes this local file (DiscardDeferredMigration) and stays on terraform.
	if err := os.MkdirAll(filepath.Dir(localDirectPath), 0o700); err != nil {
		return false, fmt.Errorf("creating local state directory for migration: %w", err)
	}
	if err := os.Rename(tempStatePath, localDirectPath); err != nil {
		return false, fmt.Errorf("writing local direct state for migration: %w", err)
	}
	if err := b.DeploymentBundle.StateDB.Open(ctx, localDirectPath, dstate.WithRecovery(true), dstate.WithWrite(false), dstate.WithDeploymentHistory(false), dstate.OpenDmsArgs{}); err != nil {
		return false, fmt.Errorf("opening migrated local state: %w", err)
	}

	// The command commits this state once approved and then FinalizeDeferredMigration cleans up
	// terraform. Report the migration now; a declined command adds a "not committed" warning.
	b.MigrationDeferred = true
	suffix := "s"
	if resourceCount == 1 {
		suffix = ""
	}
	cmdio.LogString(ctx, fmt.Sprintf("Migrated %d resource%s to direct deployment engine.", resourceCount, suffix))
	return true, nil
}

// CleanupTerraformStateAfterMigration backs up the now-superseded terraform state, remote and
// local, best-effort. Called after a MigrateDeferred command commits the direct state, so a
// later run does not pick up the stale terraform state (its resources are gone) instead of the
// direct one. Backing up the local file matters even for destroy: destroy removes the local
// resources.json, so a lingering local terraform.tfstate would otherwise win the next deploy.
func CleanupTerraformStateAfterMigration(ctx context.Context, b *bundle.Bundle) {
	BackupRemoteTerraformState(ctx, b)
	_, localTerraformPath := b.StateFilenameTerraform(ctx)
	if err := os.Rename(localTerraformPath, localTerraformPath+".backup"); err != nil && !errors.Is(err, fs.ErrNotExist) {
		log.Warnf(ctx, "automatic migration to direct engine: could not back up local terraform state: %v", err)
	}
}

// FinalizeDeferredMigration completes a deploy's MigrateDeferred migration after deployCore has
// committed the converted direct state: it cleans up the terraform state and records which
// source triggered the migration. The deploy phase calls it only once the deploy is approved
// and applied, so a declined deploy leaves the terraform state untouched. The "Migrated N
// resources" summary was already printed when the converted state was prepared. (Destroy calls
// CleanupTerraformStateAfterMigration directly - a destroy that migrates only to tear down is
// not migration adoption worth recording as a source.)
func FinalizeDeferredMigration(ctx context.Context, b *bundle.Bundle, requiredEngine engine.EngineSetting) {
	CleanupTerraformStateAfterMigration(ctx, b)
	recordAutoMigrateSource(b, requiredEngine)
}

// DiscardDeferredMigration undoes a MigrateDeploy migration whose deploy was declined: it
// removes the local direct state file and WAL the migration wrote and resets the in-memory
// state, so nothing is committed and the run stays on the terraform state. The remote
// workspace was never touched (its push is part of the approved deploy).
func DiscardDeferredMigration(ctx context.Context, b *bundle.Bundle) {
	// DiscardWrite closes and removes the WAL and resets the in-memory state.
	b.DeploymentBundle.StateDB.DiscardWrite(ctx)
	_, localDirectPath := b.StateFilenameDirect(ctx)
	if err := os.Remove(localDirectPath); err != nil && !errors.Is(err, fs.ErrNotExist) {
		log.Warnf(ctx, "could not remove local direct state after a declined migration: %v", err)
	}
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

// BackupTerraformState moves the terraform state to .backup both remotely
// (read → write .backup → delete) and locally (rename). Every step must
// succeed, so callers get an accurate error path — a stale terraform state
// left anywhere lets it win over remote direct in PullResourcesState when
// AlwaysPull is off. Missing files (both local and remote) are treated as
// no-ops, so this helper is safe to call whether or not any state exists.
// Contrast with BackupRemoteTerraformState, which only handles the remote
// half and swallows errors via log.Warnf for best-effort direct-engine
// cleanup on unrelated code paths.
func BackupTerraformState(ctx context.Context, b *bundle.Bundle) error {
	f, err := deploy.StateFiler(ctx, b)
	if err != nil {
		return err
	}
	remoteTerraformPath, localTerraformPath := b.StateFilenameTerraform(ctx)
	reader, err := f.Read(ctx, remoteTerraformPath)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("reading remote terraform state %s: %w", remoteTerraformPath, err)
	}
	if err == nil {
		defer reader.Close()
		if err := f.Write(ctx, remoteTerraformPath+".backup", reader, filer.OverwriteIfExists); err != nil {
			return fmt.Errorf("writing remote terraform backup: %w", err)
		}
		if err := f.Delete(ctx, remoteTerraformPath); err != nil {
			return fmt.Errorf("deleting remote terraform state: %w", err)
		}
	}

	if err := os.Rename(localTerraformPath, localTerraformPath+".backup"); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("renaming local terraform state to %s.backup: %w", localTerraformPath, err)
	}
	return nil
}

// convertTFStateToDirect converts the given terraform state to the direct engine state,
// returning the path to the converted state file, the number of resources
// migrated, whether any warnings were emitted, and the bundle config with
// terraform interpolation reversed (needed by the caller to run a plan against
// the converted state). Callers must ensure tfState is non-nil and has at least
// one resource ID (the empty and nil cases are handled by MigrateToDirect
// directly, since they take different commit paths). The caller is responsible
// for deleting the temp state's parent directory when it is done with the file.
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

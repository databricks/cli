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

// MigrateTerraformState converts the bundle's terraform state to a direct-engine state
// and opens b.DeploymentBundle.StateDB with it. Returns false when there is no terraform
// state to migrate (the caller then opens the direct state normally).
//
// When commit is false (plan) the converted state is opened in memory and nothing is
// written to disk or the workspace. When commit is true (deploy) the migration is
// plan-checked first; on success resources.json is written and pushed and
// terraform.tfstate is backed up, so it completes even if the deploy is a no-op.
//
// Any failure before resources.json is pushed (parsing, conversion, the empty-state
// sweep, the plan check, or the push itself) is non-fatal: a warning is emitted, false is
// returned, and the caller proceeds on the terraform engine, which is still in place. This
// keeps a failed migration from blocking a deploy that would otherwise succeed. Only a
// failure after the push (placing or opening the local state) returns an error, because
// the workspace is already committed to the direct engine by then; re-running recovers by
// pulling the committed resources.json.
func MigrateTerraformState(ctx context.Context, b *bundle.Bundle, requiredEngine engine.EngineSetting, commit bool) (bool, error) {
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

	// A terraform state with no managed resources carries nothing to migrate. On deploy,
	// sweep the empty terraform state aside so the direct engine is used from now on; on
	// plan, just open an empty direct database in memory. No resources.json is written.
	if len(tfState.IDs) == 0 && len(tfState.Attrs) == 0 {
		if commit {
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

	if commit {
		// Plan-check the converted state; if it fails, do not commit and fall back to the
		// terraform engine (the temp file is cleaned up by the deferred Remove).
		plan, err := checkPlanOnTempState(ctx, b, tempStatePath, cfg)
		if err != nil {
			b.Metrics.SetBoolValue(metrics.DirectMigratePlanError, true)
			log.Warnf(ctx, "migration to the direct engine failed its plan check; deploying on terraform this time: %v", err)
			return false, nil
		}

		// Do not migrate into a plan that would recreate an existing resource: a recreate
		// is a destroy + create and risks data loss. Fall back to terraform this run (the
		// migration retries next run, once any pending recreate has been applied).
		if recreated := recreatedResources(plan); len(recreated) > 0 {
			b.Metrics.SetBoolValue(metrics.DirectMigrateRecreatePlanned, true)
			log.Warnf(ctx, "migration to the direct engine would recreate %v; deploying on terraform this time", recreated)
			return false, nil
		}

		// Commit to the workspace by pushing resources.json (serial tf+1). This upload is
		// the hard commit: once it lands it outranks any leftover terraform state by serial,
		// so engine selection can no longer pick terraform and the cleanup in
		// finalizeLocalMigration is best-effort. If the push fails the workspace has not
		// switched engines, so deploy on terraform this time; the next deploy retries the
		// migration (the temp file is cleaned up by the deferred Remove).
		if err := pushMigrationToRemote(ctx, b, tempStatePath); err != nil {
			b.Metrics.SetBoolValue(metrics.DirectMigrateCommitError, true)
			log.Warnf(ctx, "migration to the direct engine could not be committed to the workspace; deploying on terraform this time: %v", err)
			return false, nil
		}

		// The workspace is committed to direct. Place the converted state locally (the
		// deploy about to run reads it) and best-effort clean up the terraform state. A
		// local-placement failure is still surfaced — this checkout cannot deploy without
		// it — but terraform-cleanup failures only warn.
		if err := finalizeLocalMigration(ctx, b, tempStatePath, resourceCount); err != nil {
			b.Metrics.SetBoolValue(metrics.DirectMigrateCommitError, true)
			return false, err
		}

		recordAutoMigrateSource(b, requiredEngine)

		if err := b.DeploymentBundle.StateDB.Open(ctx, localDirectPath, dstate.WithRecovery(true), dstate.WithWrite(false), dstate.WithDeploymentHistory(false), dstate.OpenDmsArgs{}); err != nil {
			return false, fmt.Errorf("migrated to the direct engine in the workspace, but opening the local direct state failed; re-run the command to deploy: %w", err)
		}
		return true, nil
	}

	// Plan (no commit): load the converted state into memory. It was just written by this
	// CLI, so it is at the current schema version and needs no migration.
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
// (destroy + create). A migration should not commit into such a plan: a recreate
// on a resource that already exists risks destroying it, whether it comes from a
// conversion that did not faithfully reproduce an immutable field or from a real
// pending config change (which terraform would recreate too). In both cases the
// safe choice is to stay on terraform this run and retry the migration later.
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

// pushMigrationToRemote commits the migration to the workspace by uploading resources.json
// (the hard commit; see pushDirectState). It first checks that the local direct state does
// not already exist, then pushes. It performs no local swap and does not touch the
// terraform state, so on failure the workspace has not switched engines and the run can
// fall back to terraform. The terraform cleanup and local swap happen afterwards in
// finalizeLocalMigration.
func pushMigrationToRemote(ctx context.Context, b *bundle.Bundle, tempStatePath string) error {
	_, localDirectPath := b.StateFilenameDirect(ctx)

	// A stat error other than "not exist" (e.g. permission denied) is not
	// "file is missing"; treat it as a hard failure to avoid renaming over
	// something we couldn't read.
	if _, err := os.Stat(localDirectPath); err == nil {
		return fmt.Errorf("state file %s already exists", localDirectPath)
	} else if !errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("stat %s: %w", localDirectPath, err)
	}

	if err := pushDirectState(ctx, b, tempStatePath); err != nil {
		return fmt.Errorf("pushing direct state to workspace: %w", err)
	}
	return nil
}

// finalizeLocalMigration places the converted state locally and cleans up the terraform
// state after pushMigrationToRemote committed resources.json to the workspace.
//
// The local direct-state placement must succeed: the deploy about to run reads it, so a
// failure is returned. The terraform cleanup (remote via BackupRemoteTerraformState and
// the local terraform.tfstate rename) is best-effort — resources.json already has the
// higher serial (tf+1) and outranks any leftover terraform state in engine selection, so
// a cleanup failure only leaves a harmless stale file and is logged rather than returned.
func finalizeLocalMigration(ctx context.Context, b *bundle.Bundle, tempStatePath string, resourceCount int) error {
	_, localDirectPath := b.StateFilenameDirect(ctx)

	if err := os.MkdirAll(filepath.Dir(localDirectPath), 0o700); err != nil {
		return fmt.Errorf("migrated to the direct engine in the workspace, but creating the local state directory failed; re-run the command to deploy: %w", err)
	}
	if err := os.Rename(tempStatePath, localDirectPath); err != nil {
		return fmt.Errorf("migrated to the direct engine in the workspace, but writing the local direct state failed; re-run the command to deploy: %w", err)
	}

	// Best-effort terraform cleanup, remote and local.
	BackupRemoteTerraformState(ctx, b)
	_, localTerraformPath := b.StateFilenameTerraform(ctx)
	if err := os.Rename(localTerraformPath, localTerraformPath+".backup"); err != nil && !errors.Is(err, fs.ErrNotExist) {
		log.Warnf(ctx, "automatic migration to direct engine: could not back up local terraform state: %v", err)
	}

	suffix := "s"
	if resourceCount == 1 {
		suffix = ""
	}
	cmdio.LogString(ctx, fmt.Sprintf("Migrated %d resource%s to direct deployment engine.", resourceCount, suffix))
	return nil
}

// pushDirectState uploads the direct-engine state file (resources.json) to the workspace.
// The caller passes the temp state produced by the dry-run. This upload is the migration's
// hard commit: resources.json carries serial tf+1, so once it lands it outranks any
// leftover terraform state and the terraform cleanup becomes best-effort (see
// finalizeLocalMigration). It does not touch the terraform state itself.
func pushDirectState(ctx context.Context, b *bundle.Bundle, localPath string) error {
	f, err := deploy.StateFiler(ctx, b)
	if err != nil {
		return err
	}

	remoteDirectPath, _ := b.StateFilenameDirect(ctx)
	local, err := os.Open(localPath)
	if err != nil {
		return err
	}
	defer local.Close()

	return f.Write(ctx, remoteDirectPath, local, filer.CreateParentDirectories, filer.OverwriteIfExists)
}

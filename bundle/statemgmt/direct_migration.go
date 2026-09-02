package statemgmt

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/engine"
	"github.com/databricks/cli/bundle/config/mutator/resourcemutator"
	"github.com/databricks/cli/bundle/deploy"
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

// warnPrefix labels warnings emitted by the post-deploy dry-run so they are not
// confused with warnings from the user-invoked `bundle migrate` command.
const warnPrefix = "post-deploy dry-run migration to direct: "

// feedbackNotice is appended after the dry-run warnings to reassure the user the
// deploy is unaffected and to ask them to report the warnings.
const feedbackNotice = `The warnings above are from a dry-run migration to the direct deployment engine (https://docs.databricks.com/aws/en/dev-tools/bundles/direct).
Your deployment is not affected and works normally, but you may experience these issues when migrating to the direct deployment engine.
Please forward these warnings to dabs-feedback@databricks.com`

// autoMigrateStoppedNotice is emitted when the direct engine is selected but the
// dry-run migration surfaced errors or warnings, so the automatic post-deploy
// migration is skipped.
const autoMigrateStoppedNotice = `Direct engine was selected but the dry-run migration reported issues; automatic migration to the direct deployment engine is stopped. Address the issues above or run "databricks bundle deployment migrate" manually.`

// MigrateToDirect performs a dry-run migration of the just-deployed terraform
// state to the direct engine and records the outcome in deploy telemetry.
//
// The converted state is written to a temporary file. If the dry-run is clean
// and requestedEngine resolves to "direct" (which is the default, and can also
// be set explicitly via bundle.engine or the DATABRICKS_BUNDLE_ENGINE env var),
// the temp state is committed (renamed to resources.json, terraform.tfstate is
// backed up, and the new state is pushed to the workspace). Otherwise the temp
// state is deleted and only telemetry is recorded. Any failure is surfaced as a
// warning so it never fails a deploy that already succeeded.
func MigrateToDirect(ctx context.Context, b *bundle.Bundle, requestedEngine engine.EngineSetting) {
	_, localTerraformPath := b.StateFilenameTerraform(ctx)
	tfState, err := migrate.ParseTFStateFull(ctx, localTerraformPath)
	if err != nil {
		log.Warnf(ctx, "%sfailed to parse terraform state: %v", warnPrefix, err)
		if requestedEngine.Type == engine.EngineDirect {
			b.Metrics.SetBoolValue(metrics.DirectMigrateError, true)
			log.Warnf(ctx, "%s", autoMigrateStoppedNotice)
		} else {
			b.Metrics.SetBoolValue(metrics.DirectDryMigrateSuccess, false)
		}
		return
	}

	if tfState == nil {
		// No terraform state file to migrate; nothing to do either way.
		return
	}

	// A terraform.tfstate that has no databricks_* resources AND no managed
	// resources of any kind has no state to migrate. Gate the sweep on both:
	// Attrs is empty when the file has zero managed resources; IDs is empty
	// when nothing maps to a known DABs group. Sweeping when Attrs is
	// non-empty (unknown TF resource types in the state) would destroy state
	// the CLI doesn't recognize, so restrict the sweep to the truly-empty
	// case and let the populated path handle everything else.
	if len(tfState.IDs) == 0 && len(tfState.Attrs) == 0 {
		if requestedEngine.Type != engine.EngineDirect {
			recordDryRunNoop(b, requestedEngine)
			return
		}
		cmdio.LogString(ctx, "Removing empty terraform state; direct engine will be used on the next deploy (selected via "+requestedEngine.Source+")...")
		if err := backupTerraformState(ctx, b); err != nil {
			b.Metrics.SetBoolValue(metrics.DirectMigrateCommitError, true)
			log.Warnf(ctx, "automatic migration to direct engine failed: %v", err)
			return
		}
		recordAutoMigrateSource(b, requestedEngine)
		return
	}

	tempStatePath, resourceCount, hasWarnings, cfg, err := convertTFStateToDirect(ctx, b, tfState)
	if tempStatePath != "" {
		// The temp file sits next to the real resources.json path (same
		// filesystem, so commitMigration's os.Rename works even when
		// os.TempDir() is on a different volume). Clean up the temp file
		// and its WAL sibling — commitMigration renames the state file out
		// of the way on success, so these Removes are no-ops in that case.
		defer func() {
			_ = os.Remove(tempStatePath)
			_ = os.Remove(tempStatePath + ".wal")
		}()
	}

	if err != nil {
		log.Warnf(ctx, "%s%v", warnPrefix, err)
	}
	if hasWarnings || err != nil {
		log.Warnf(ctx, "%s", feedbackNotice)
	}

	// The direct engine was not selected (the user opted out with
	// engine: terraform) — the conversion was only a dry run for fleet-wide
	// telemetry, so record dry-run outcome only.
	if requestedEngine.Type != engine.EngineDirect {
		b.Metrics.SetBoolValue(metrics.DirectDryMigrateSuccess, err == nil)
		b.Metrics.SetBoolValue(metrics.DirectDryMigrateWarnings, hasWarnings)
		return
	}

	// From here on, direct is the engine to migrate to: use the migrate_* telemetry keys.
	if err != nil {
		b.Metrics.SetBoolValue(metrics.DirectMigrateError, true)
	}
	if hasWarnings {
		b.Metrics.SetBoolValue(metrics.DirectMigrateWarnings, true)
	}

	if err != nil || hasWarnings {
		log.Warnf(ctx, "%s", autoMigrateStoppedNotice)
		return
	}

	if planErr := checkPlanOnTempState(ctx, b, tempStatePath, cfg); planErr != nil {
		log.Warnf(ctx, "%s%v", warnPrefix, planErr)
		log.Warnf(ctx, "%s", feedbackNotice)
		b.Metrics.SetBoolValue(metrics.DirectMigratePlanError, true)
		log.Warnf(ctx, "%s", autoMigrateStoppedNotice)
		return
	}

	cmdio.LogString(ctx, "Migrating state to direct deployment engine (selected via "+requestedEngine.Source+")...")

	if err := commitMigration(ctx, b, tempStatePath, resourceCount); err != nil {
		b.Metrics.SetBoolValue(metrics.DirectMigrateCommitError, true)
		log.Warnf(ctx, "automatic migration to direct engine failed: %v", err)
		return
	}

	recordAutoMigrateSource(b, requestedEngine)
}

// checkPlanOnTempState opens the migrated state at tempStatePath in read mode,
// runs a full plan against it, and returns a non-nil error if the plan fails.
// Individual planning errors are emitted as warnings with warnPrefix so they
// are visible without failing the deploy. The plan is run in an isolated
// context so its diagnostics do not affect the deploy's own error state.
func checkPlanOnTempState(ctx context.Context, b *bundle.Bundle, tempStatePath string, cfg *config.Root) error {
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
	if err := planBundle.StateDB.Open(planCtx, tempStatePath, false, false); err != nil {
		return fmt.Errorf("opening migrated state for plan check: %w", err)
	}

	if _, err := planBundle.CalculatePlan(planCtx, b.WorkspaceClient(ctx), cfg); err != nil {
		return err
	}
	if logdiag.HasError(planCtx) {
		return errors.New("plan check failed")
	}
	return nil
}

// recordDryRunNoop records dry-run telemetry for a no-op case (no state, or
// state with no managed resources) when direct was NOT selected. On the
// migrating paths the caller uses direct_migrate_* keys instead.
func recordDryRunNoop(b *bundle.Bundle, requestedEngine engine.EngineSetting) {
	if requestedEngine.Type == engine.EngineDirect {
		return
	}
	b.Metrics.SetBoolValue(metrics.DirectDryMigrateSuccess, true)
	b.Metrics.SetBoolValue(metrics.DirectDryMigrateWarnings, false)
}

// recordAutoMigrateSource sets exactly one of the migrated-via-* telemetry
// keys. requestedEngine.Type may resolve to direct from the config, the env var,
// or the default (config wins over env in ResolveEngineSetting). ConfigType is
// set only when the config populated the setting, so it's the correct signal for
// "was this a durable opt-in?" — env-only opt-ins are the ones with
// ConfigType == EngineNotSet and IsDefault false.
func recordAutoMigrateSource(b *bundle.Bundle, requestedEngine engine.EngineSetting) {
	switch {
	case requestedEngine.IsDefault:
		b.Metrics.SetBoolValue(metrics.DirectAutoMigrateViaDefault, true)
	case requestedEngine.ConfigType == engine.EngineDirect:
		b.Metrics.SetBoolValue(metrics.DirectAutoMigrateViaConfig, true)
	default:
		b.Metrics.SetBoolValue(metrics.DirectAutoMigrateViaEnv, true)
	}
}

// backupTerraformState moves the terraform state to .backup both remotely
// (read → write .backup → delete) and locally (rename). Every step must
// succeed, so callers get an accurate error path — a stale terraform state
// left anywhere lets it win over remote direct in PullResourcesState when
// AlwaysPull is off. Missing files (both local and remote) are treated as
// no-ops, so this helper is safe to call whether or not any state exists.
// Contrast with BackupRemoteTerraformState, which only handles the remote
// half and swallows errors via log.Warnf for best-effort direct-engine
// cleanup on unrelated code paths.
func backupTerraformState(ctx context.Context, b *bundle.Bundle) error {
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

// commitMigration finalizes the dry-run migration by pushing the direct state
// to the workspace, moving the local state files into place, and backing up
// the remote terraform state. Remote push happens FIRST: if we swapped local
// state and then failed to push, this machine would prefer direct state while
// the workspace still has terraform state, so other machines would diverge.
func commitMigration(ctx context.Context, b *bundle.Bundle, tempStatePath string, resourceCount int) error {
	_, localTerraformPath := b.StateFilenameTerraform(ctx)
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

	// Remote is now authoritative for direct engine; make local match. Local
	// updates must succeed so the next deploy from this machine picks direct
	// state directly — a stale local terraform.tfstate would win over the
	// remote direct state whenever AlwaysPull is off. Report a commit error
	// on failure so telemetry reflects what actually happened here (the
	// migration is complete on the workspace but not on this checkout).
	if err := os.MkdirAll(filepath.Dir(localDirectPath), 0o700); err != nil {
		return fmt.Errorf("workspace migrated but creating local state directory failed: %w", err)
	}
	if err := os.Rename(tempStatePath, localDirectPath); err != nil {
		return fmt.Errorf("workspace migrated but writing local direct state failed: %w", err)
	}
	if err := os.Rename(localTerraformPath, localTerraformPath+".backup"); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("workspace migrated but backing up local terraform state failed: %w", err)
	}

	suffix := "s"
	if resourceCount == 1 {
		suffix = ""
	}
	cmdio.LogString(ctx, fmt.Sprintf("Migrated %d resource%s to direct deployment engine.", resourceCount, suffix))
	return nil
}

// pushDirectState uploads the direct-engine state file to the workspace and
// moves the remote terraform state aside so it is no longer authoritative.
// The caller passes the file whose contents to upload — this is the temp
// state produced by the dry-run, uploaded before it is renamed into place
// locally so the workspace becomes authoritative first.
//
// Backup/delete errors on the remote terraform state are fatal here: if the
// direct state landed but the terraform state stayed, the workspace has two
// authoritative files, and an older CLI (or `validateStates`) will refuse to
// use them. Fail loudly so the caller can record telemetry and warn the user.
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

	if err := f.Write(ctx, remoteDirectPath, local, filer.CreateParentDirectories, filer.OverwriteIfExists); err != nil {
		return err
	}

	// Move the remote terraform state to .backup so a future deploy from an
	// older CLI does not race the two state files.
	remoteTerraformPath, _ := b.StateFilenameTerraform(ctx)
	reader, err := f.Read(ctx, remoteTerraformPath)
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("reading remote terraform state %s: %w", remoteTerraformPath, err)
	}
	defer reader.Close()

	if err := f.Write(ctx, remoteTerraformPath+".backup", reader, filer.OverwriteIfExists); err != nil {
		return fmt.Errorf("writing remote terraform backup: %w", err)
	}

	if err := f.Delete(ctx, remoteTerraformPath); err != nil {
		return fmt.Errorf("deleting remote terraform state: %w", err)
	}

	return nil
}

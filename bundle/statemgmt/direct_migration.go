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

// autoMigrateStoppedNotice is emitted when the user opted in to the direct
// engine but the dry-run migration surfaced errors or warnings, so the
// automatic post-deploy migration is skipped.
const autoMigrateStoppedNotice = `Direct engine was requested but the dry-run migration reported issues; automatic migration to the direct deployment engine is stopped. Address the issues above or run "databricks bundle deployment migrate" manually.`

// MigrateToDirect performs a dry-run migration of the just-deployed terraform
// state to the direct engine and records the outcome in deploy telemetry.
//
// The converted state is written to a temporary file. If the dry-run is clean
// and requestedEngine resolves to "direct" (via bundle.engine or the
// DATABRICKS_BUNDLE_ENGINE env var), the temp state is committed (renamed to
// resources.json, terraform.tfstate is backed up, and the new state is pushed
// to the workspace). Otherwise the temp state is deleted and only telemetry
// is recorded. Any failure is surfaced as a warning so it never fails a
// deploy that already succeeded.
func MigrateToDirect(ctx context.Context, b *bundle.Bundle, requestedEngine engine.EngineSetting) {
	// Announce the migration attempt before running it so the "will try" hint
	// only appears when a migration is actually about to happen (not on
	// non-deploy commands that also resolve the engine setting).
	if requestedEngine.Type == engine.EngineDirect {
		cmdio.LogString(ctx, "Attempting to migrate state to direct deployment engine (opted in via "+requestedEngine.Source+")...")
	}

	tempStatePath, resourceCount, hasWarnings, err := dryRunMigrate(ctx, b)
	if tempStatePath != "" {
		// commitMigration renames the state file out of this dir, but the dir
		// itself and any leftover files (WAL, etc.) still need cleanup.
		defer os.RemoveAll(filepath.Dir(tempStatePath))
	}

	if err != nil {
		log.Warnf(ctx, "%s%v", warnPrefix, err)
	}
	if hasWarnings || err != nil {
		log.Warnf(ctx, "%s", feedbackNotice)
	}

	// The user did not opt in to the direct engine — the conversion was only
	// a dry run for fleet-wide telemetry, so record dry-run outcome only.
	if requestedEngine.Type != engine.EngineDirect {
		b.Metrics.SetBoolValue(metrics.DirectDryMigrateSuccess, err == nil)
		b.Metrics.SetBoolValue(metrics.DirectDryMigrateWarnings, hasWarnings)
		return
	}

	// From here on, the user opted in: use the migrate_* telemetry keys.
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

	if tempStatePath == "" {
		// Nothing to migrate (no terraform state file).
		return
	}

	if err := commitMigration(ctx, b, tempStatePath, resourceCount); err != nil {
		b.Metrics.SetBoolValue(metrics.DirectMigrateCommitError, true)
		log.Warnf(ctx, "automatic migration to direct engine failed: %v", err)
		return
	}

	// Record the opt-in source so we can tell how many auto-migrations came
	// from a committed config change vs. a transient env-var override.
	if requestedEngine.ConfigType == engine.EngineDirect {
		b.Metrics.SetBoolValue(metrics.DirectAutoMigrateViaConfig, true)
	} else {
		b.Metrics.SetBoolValue(metrics.DirectAutoMigrateViaEnv, true)
	}
}

// dryRunMigrate converts the local terraform state to the direct engine state,
// returning the path to the converted state file (empty if there was nothing
// to migrate), the number of resources migrated, and whether any warnings were
// emitted. The caller is responsible for deleting the temp state's parent
// directory when it is done with the file.
func dryRunMigrate(ctx context.Context, b *bundle.Bundle) (string, int, bool, error) {
	_, localTerraformPath := b.StateFilenameTerraform(ctx)
	tfState, err := migrate.ParseTFStateFull(ctx, localTerraformPath)
	if err != nil {
		return "", 0, false, fmt.Errorf("failed to parse terraform state: %w", err)
	}

	// ParseTFStateFull returns nil when the terraform state file doesn't exist
	// (e.g. first deploy with no resources); nothing to migrate, trivially OK.
	if tfState == nil {
		return "", 0, false, nil
	}

	// Write the converted state to a temp dir. If the dry-run is clean and the
	// caller commits the migration, the state file is moved into place; otherwise
	// the caller removes the whole temp dir (along with the WAL created below).
	tempDir, err := os.MkdirTemp("", "databricks-direct-migration-")
	if err != nil {
		return "", 0, false, fmt.Errorf("failed to create temp dir: %w", err)
	}
	tempStatePath := filepath.Join(tempDir, "resources.json")
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
		return tempStatePath, resourceCount, false, errors.New("failed to apply secret scope fixups")
	}

	// b.Config has been modified by terraform.Interpolate which converts bundle-style
	// references (${resources.pipelines.x.id}) to terraform-style (${databricks_pipeline.x.id}).
	// BuildStateFromTF expects ${resources.*} references, so reverse the interpolation first.
	uninterpolatedRoot, err := reverseInterpolate(b.Config.Value())
	if err != nil {
		return tempStatePath, resourceCount, false, fmt.Errorf("failed to reverse interpolation: %w", err)
	}

	var uninterpolatedConfig config.Root
	err = uninterpolatedConfig.Mutate(func(_ dyn.Value) (dyn.Value, error) {
		return uninterpolatedRoot, nil
	})
	if err != nil {
		return tempStatePath, resourceCount, false, fmt.Errorf("failed to create uninterpolated config: %w", err)
	}

	adapters, err := dresources.InitAll(nil)
	if err != nil {
		return tempStatePath, resourceCount, false, err
	}

	if err := stateDB.UpgradeToWrite(); err != nil {
		return tempStatePath, resourceCount, false, fmt.Errorf("upgrading state for apply: %w", err)
	}

	// warnPrefix labels the conversion's warnings as coming from the background dry run.
	hasWarnings, err := migrate.BuildStateFromTF(ctx, &uninterpolatedConfig, adapters, &stateDB, tfState.Attrs, tfState.IDs, warnPrefix)
	if err != nil {
		return tempStatePath, resourceCount, hasWarnings, err
	}

	if _, err := stateDB.Finalize(ctx); err != nil {
		return tempStatePath, resourceCount, hasWarnings, err
	}

	// BuildStateFromTF reports some failures via logdiag instead of returning an error.
	if logdiag.HasError(ctx) {
		return tempStatePath, resourceCount, hasWarnings, errors.New("state conversion failed")
	}

	return tempStatePath, resourceCount, hasWarnings, nil
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

	// Remote is now authoritative for direct engine; make local match.
	if err := os.MkdirAll(filepath.Dir(localDirectPath), 0o700); err != nil {
		return fmt.Errorf("creating state directory: %w", err)
	}

	if err := os.Rename(tempStatePath, localDirectPath); err != nil {
		return fmt.Errorf("renaming %s to %s: %w", tempStatePath, localDirectPath, err)
	}

	if err := os.Rename(localTerraformPath, localTerraformPath+".backup"); err != nil {
		// Not fatal — the direct state is already in place with a bumped serial,
		// so future deploys will use it. A leftover terraform.tfstate is
		// harmless (PullResourcesState picks the state with the highest serial).
		log.Warnf(ctx, "could not back up terraform state at %s: %v", localTerraformPath, err)
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
	BackupRemoteTerraformState(ctx, b)
	return nil
}

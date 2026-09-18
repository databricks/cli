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
// terraform.tfstate is backed up, so it completes even if the deploy is a no-op. If the
// plan check fails the migration is not committed and a warning is emitted; the caller
// then proceeds on the terraform engine, which is still available.
func MigrateTerraformState(ctx context.Context, b *bundle.Bundle, commit bool) (bool, error) {
	_, localTerraformPath := b.StateFilenameTerraform(ctx)
	tfState, err := migrate.ParseTFStateFull(ctx, localTerraformPath)
	if err != nil {
		return false, fmt.Errorf("parsing terraform state: %w", err)
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
				return false, err
			}
		}
		b.DeploymentBundle.StateDB.OpenWithData(localDirectPath, dstate.NewDatabase(tfState.Lineage, tfState.Serial+1))
		return true, nil
	}

	tempStatePath, resourceCount, _, cfg, err := convertTFStateToDirect(ctx, b, tfState)
	cleanupTemp := true
	if tempStatePath != "" {
		defer func() {
			if cleanupTemp {
				_ = os.Remove(tempStatePath)
				_ = os.Remove(tempStatePath + ".wal")
			}
		}()
	}
	if err != nil {
		return false, err
	}

	if commit {
		// Plan-check the converted state; if it fails, do not commit and fall back to the
		// terraform engine (the temp file is cleaned up by the deferred Remove).
		if err := checkPlanOnTempState(ctx, b, tempStatePath, cfg); err != nil {
			log.Warnf(ctx, "migration to the direct engine failed its plan check; deploying on terraform this time: %v", err)
			return false, nil
		}
		// Commit: move resources.json into place, push it, and back up terraform.tfstate.
		cleanupTemp = false
		if err := commitMigration(ctx, b, tempStatePath, resourceCount); err != nil {
			return false, err
		}
		if err := b.DeploymentBundle.StateDB.Open(ctx, localDirectPath, dstate.WithRecovery(true), dstate.WithWrite(false), dstate.WithDeploymentHistory(false), dstate.OpenDmsArgs{}); err != nil {
			return false, fmt.Errorf("opening migrated state: %w", err)
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

	// This plan is not created with the deployment history feature enabled,
	// so we can safely pass false for withDeploymentHistory.
	if err := planBundle.StateDB.Open(planCtx, tempStatePath, false, false, dstate.WithDeploymentHistory(false), dstate.OpenDmsArgs{}); err != nil {
		return fmt.Errorf("opening migrated state for plan check: %w", err)
	}

	_, err := planBundle.CalculatePlan(planCtx, b.WorkspaceClient(ctx), cfg)
	return err
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

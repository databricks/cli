package snapshot

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/bundle/direct/dresources"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/iamutil"
	"github.com/databricks/cli/libs/snapshot"
	"github.com/databricks/databricks-sdk-go/apierr"
)

// fileLimitWarning is the file count above which immutable folder deployments may fail.
const fileLimitWarning = 1000

type snapshotUpload struct {
	// clean is true for a fresh plan or deploy: the mutator discards zips staged by a
	// previous run (so content-addressed "<hash>.zip" files don't accumulate) and
	// stages the current one. It is false when applying a pre-existing plan
	// (deploy --plan): the plan already records the resource and the zip staged when it
	// was produced, so the mutator reuses that staged file instead of rebuilding it.
	clean bool
}

type PlanUploadOptions struct {
	Clean bool
}

// PlanUpload returns a mutator that registers the immutable snapshot as an internal
// resource. When Clean is true it discards zips staged by a previous run, builds the
// bundle zip, stages it under the bundle's local state directory as "<hash>.zip", and
// records the path on the resource; the staged file is uploaded when the resource is
// created on apply. Pass Clean=false when applying a pre-existing plan: the plan
// already carries the resource and its staged zip, so the mutator reuses them.
func PlanUpload(opts PlanUploadOptions) bundle.Mutator {
	return &snapshotUpload{clean: opts.Clean}
}

func (m *snapshotUpload) Name() string {
	return "snapshot.Upload"
}

func (m *snapshotUpload) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	// Applying a pre-existing plan: the plan already records the snapshot resource and
	// the path of the zip staged when the plan was produced (InitForApply restores that
	// state and DoCreate reads the staged file). Reuse it instead of rebuilding.
	if !m.clean {
		return nil
	}

	uploader, err := snapshot.NewSnapshotClient(b.WorkspaceClient(ctx))
	if err != nil {
		return diag.FromErr(err)
	}

	remoteRoot, err := uploader.GetSnapshotRootPath(ctx)
	if err != nil {
		return diag.FromErr(err)
	}

	if b.Config.Resources.Snapshots == nil {
		b.Config.Resources.Snapshots = make(map[string]*resources.Snapshot)
	}
	if _, ok := b.Config.Resources.Snapshots[resources.SnapshotResourceKey]; ok {
		return nil
	}

	// Check the previous snapshot before doing any work, so a deploy onto a snapshot that was
	// modified outside the bundle fails before building a zip.
	generation, err := resolveGeneration(ctx, b, uploader, remoteRoot)
	if err != nil {
		return diag.FromErr(err)
	}

	zipContent, fileCount, err := BundleZip(ctx, b)
	if err != nil {
		return diag.FromErr(fmt.Errorf("failed to build snapshot zip: %w", err))
	}

	// Discard zips staged by a previous run so content-addressed "<hash>.zip" files
	// don't accumulate.
	if err := os.RemoveAll(b.GetLocalStateDir(ctx, "snapshots")); err != nil {
		return diag.FromErr(fmt.Errorf("failed to clean snapshot dir: %w", err))
	}
	dir, err := b.LocalStateDir(ctx, "snapshots")
	if err != nil {
		return diag.FromErr(err)
	}
	zipPath := filepath.Join(dir, snapshot.HashFromContent(zipContent)+".zip")
	if err := os.WriteFile(zipPath, zipContent, 0o600); err != nil {
		return diag.FromErr(fmt.Errorf("failed to write snapshot zip: %w", err))
	}

	b.Config.Resources.Snapshots[resources.SnapshotResourceKey] = &resources.Snapshot{
		BundleID:   BundleID(b),
		ACL:        BuildACL(b),
		CanManage:  BuildCanManage(b),
		RemoteRoot: remoteRoot,
		ZipPath:    filepath.ToSlash(zipPath),
		Generation: generation,
	}

	var diags diag.Diagnostics
	if fileCount > fileLimitWarning {
		diags = append(diags, diag.Warningf(
			"immutable folder deployment may not work correctly: bundle contains %d files (limit is %d)",
			fileCount, fileLimitWarning,
		)...)
	}

	return diags
}

// BundleID returns a stable hash that identifies the bundle deployment.
// It is derived deterministically from workspace.state_path, which is the
// canonical unique identifier for a deployment (name, target, and workspace root
// are all encoded in it). Two bundles with the same name and target but different
// workspace.state_path values get distinct IDs.
func BundleID(b *bundle.Bundle) string {
	// Use normal sha256 without the namespace.
	hash := sha256.Sum256([]byte(b.Config.Workspace.StatePath))
	return hex.EncodeToString(hash[:])
}

// BuildACL constructs the access_control_list for the snapshot upload.
// It grants CAN_READ to the current user and to every principal listed in the
// top-level permissions section of the bundle config.
func BuildACL(b *bundle.Bundle) []snapshot.ACLEntry {
	acl := []snapshot.ACLEntry{
		{UserName: b.Config.Workspace.CurrentUser.UserName, PermissionLevel: "CAN_READ"},
	}
	for _, p := range b.Config.Permissions {
		acl = append(acl, snapshot.ACLEntry{
			UserName:             p.UserName,
			GroupName:            p.GroupName,
			ServicePrincipalName: p.ServicePrincipalName,
			PermissionLevel:      "CAN_READ",
		})
	}
	return acl
}

// BuildCanManage constructs can_manage_principals for the snapshot upload: the principals that
// may break the glass on the snapshot. Those are the deploying identity plus everyone granted
// CAN_MANAGE in the top-level permissions section. The deploying identity is always included
// because the API rejects an empty list, and it is the one principal always available. The
// recommended permissions section names it with CAN_MANAGE as well, so skip it in the loop
// rather than sending it twice.
func BuildCanManage(b *bundle.Bundle) []snapshot.ManagePrincipal {
	currentUser := b.Config.Workspace.CurrentUser.User
	var canManage []snapshot.ManagePrincipal
	if iamutil.IsServicePrincipal(currentUser) {
		canManage = append(canManage, snapshot.ManagePrincipal{
			ServicePrincipalName: currentUser.UserName,
		})
	} else {
		canManage = append(canManage, snapshot.ManagePrincipal{
			UserName: currentUser.UserName,
		})
	}
	for _, p := range b.Config.Permissions {
		if p.Level != "CAN_MANAGE" {
			continue
		}
		canManage = append(canManage, snapshot.ManagePrincipal{
			UserName:             p.UserName,
			GroupName:            p.GroupName,
			ServicePrincipalName: p.ServicePrincipalName,
		})
	}
	return canManage
}

// resolveGeneration asks the backend whether the snapshot deployed last time was modified
// outside of the bundle, and returns the generation the new snapshot should use:
//   - the previous generation while the snapshot is untouched, so a path reached by an earlier
//     recovery keeps being reused;
//   - the previous generation + 1 when it was modified and --force was given, which moves the
//     deployment to a fresh path;
//   - an error when it was modified and --force was not given.
//
// The check has to happen here rather than in the resource's DoRead: the generation decides
// the snapshot's path, and resources referencing ${...full_path} resolve it from the desired
// state computed before any resource is read (see LookupReferencePreDeploy). Erroring unless
// --force also has no equivalent in the resource layer. CheckDashboardsModifiedRemotely is the
// same shape: read state, ask the API, refuse unless forced.
//
// On the first deploy (no snapshot in state) the generation is 0.
func resolveGeneration(ctx context.Context, b *bundle.Bundle, uploader *snapshot.SnapshotClient, remoteRoot string) (int, error) {
	// The deploy/plan pipeline opens the state DB for read before this runs, but callers that
	// exercise PlanUpload in isolation (unit tests) may not. No open state means no previous
	// snapshot to check, which is the first-deploy case.
	if !b.DeploymentBundle.StateDB.IsOpen() {
		return 0, nil
	}

	entry, ok := b.DeploymentBundle.StateDB.GetResourceEntry(resources.SnapshotKey)
	if !ok || len(entry.State) == 0 {
		return 0, nil
	}

	// Decode into the state type the engine persists, so the field names cannot drift apart.
	var prev dresources.SnapshotState
	if err := json.Unmarshal(entry.State, &prev); err != nil {
		return 0, fmt.Errorf("reading previous snapshot state: %w", err)
	}
	if prev.RelativePath == "" {
		return 0, nil
	}

	// Compose the content path from relative_path rather than reading full_path: relative_path
	// already encodes the previous generation and means the same thing in every CLI version,
	// while full_path from a CLI that predates the content subfolder points one level too high.
	contentPath := path.Join(remoteRoot, prev.RelativePath, snapshot.ContentSubdir)

	status, err := uploader.InspectSnapshot(ctx, contentPath)
	// Nothing to protect when no snapshot is there: the deployment predates the content
	// subfolder, or the snapshot was deleted out of band, or an earlier deploy never created it.
	// Failing here would also be a dead end, since this runs before the --force check and so
	// could not be forced past.
	if errors.Is(err, apierr.ErrNotFound) {
		return prev.Generation, nil
	}
	if err != nil {
		return 0, err
	}
	if !status.Dirty {
		return prev.Generation, nil
	}
	if !b.Config.Bundle.Force {
		return 0, fmt.Errorf("the immutable snapshot of the last deployment was modified outside of the bundle (break glass):\n  %s\nTo deploy a new snapshot and point resources at it, use --force", contentPath)
	}
	return prev.Generation + 1, nil
}

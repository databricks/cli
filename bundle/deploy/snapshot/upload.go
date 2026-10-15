package snapshot

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"

	"github.com/databricks/cli/libs/utils"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/snapshot"
)

// fileLimitWarning is the file count above which immutable folder deployments may fail.
const fileLimitWarning = 1000

type snapshotUpload struct {
	// clean is true for a fresh plan or deploy: the mutator discards zips staged by a
	// previous run (so content-addressed "<hash>.zip" files don't accumulate) and
	// stages the current one. It is false when applying a pre-existing plan
	// (deploy --plan): the mutator reuses the zip staged when the plan was produced
	// instead of rebuilding it.
	clean bool
}

type PlanUploadOptions struct {
	Clean bool
}

// PlanUpload returns a mutator that registers the immutable snapshot as an internal
// resource. When Clean is true it discards zips staged by a previous run, builds the
// bundle zip, stages it under the bundle's local state directory as "<hash>.zip", and
// records the path on the resource; the staged file is uploaded when the resource is
// created on apply. Pass Clean=false when applying a pre-existing plan: the mutator reuses the
// zip staged when the plan was produced. Either way the resource is registered in the config,
// so its path is known to the rest of the deploy before the snapshot is uploaded.
func PlanUpload(opts PlanUploadOptions) bundle.Mutator {
	return &snapshotUpload{clean: opts.Clean}
}

func (m *snapshotUpload) Name() string {
	return "snapshot.Upload"
}

func (m *snapshotUpload) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
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

	var zipPath string
	var diags diag.Diagnostics
	if m.clean {
		zipPath, diags, err = stageZip(ctx, b)
	} else {
		zipPath, err = stagedZipPath(ctx, b)
	}
	if err != nil {
		return diags.Extend(diag.FromErr(err))
	}

	b.Config.Resources.Snapshots[resources.SnapshotResourceKey] = &resources.Snapshot{
		BundleID:   BundleID(b),
		ACL:        BuildACL(b),
		RemoteRoot: remoteRoot,
		ZipPath:    filepath.ToSlash(zipPath),
	}
	return diags
}

// stageZip builds the bundle zip and stages it under the bundle's local state directory as
// "<hash>.zip", discarding zips staged by a previous run so content-addressed files don't
// accumulate.
func stageZip(ctx context.Context, b *bundle.Bundle) (string, diag.Diagnostics, error) {
	zipContent, fileCount, err := BundleZip(ctx, b)
	if err != nil {
		return "", nil, fmt.Errorf("failed to build snapshot zip: %w", err)
	}

	if err := os.RemoveAll(b.GetLocalStateDir(ctx, "snapshots")); err != nil {
		return "", nil, fmt.Errorf("failed to clean snapshot dir: %w", err)
	}
	dir, err := b.LocalStateDir(ctx, "snapshots")
	if err != nil {
		return "", nil, err
	}
	zipPath := filepath.Join(dir, snapshot.HashFromContent(zipContent)+".zip")
	if err := os.WriteFile(zipPath, zipContent, 0o600); err != nil {
		return "", nil, fmt.Errorf("failed to write snapshot zip: %w", err)
	}

	var diags diag.Diagnostics
	if fileCount > fileLimitWarning {
		diags = diag.Warningf(
			"immutable folder deployment may not work correctly: bundle contains %d files (limit is %d)",
			fileCount, fileLimitWarning,
		)
	}
	return zipPath, diags, nil
}

// stagedZipPath returns the zip staged when the plan being applied was produced. Each fresh
// plan clears the directory before staging, so it holds exactly that one zip.
func stagedZipPath(ctx context.Context, b *bundle.Bundle) (string, error) {
	zips, err := filepath.Glob(filepath.Join(b.GetLocalStateDir(ctx, "snapshots"), "*.zip"))
	if err != nil {
		return "", err
	}
	if len(zips) != 1 {
		return "", fmt.Errorf("expected one staged snapshot zip, found %d; run 'bundle plan' again", len(zips))
	}
	return zips[0], nil
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
		{UserName: utils.Deref(b.Config.Workspace.CurrentUser.UserName), PermissionLevel: "CAN_READ"},
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

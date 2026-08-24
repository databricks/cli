package snapshot

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/snapshot"
	"github.com/google/uuid"
)

// fileLimitWarning is the file count above which immutable folder deployments may fail.
const fileLimitWarning = 1000

type snapshotUpload struct {
	// clean discards previously staged zips before staging the current one. It must
	// be false when applying a pre-existing plan (deploy --plan): that plan's
	// zip_path points at a file staged when the plan was produced, which cleaning
	// would delete out from under DoCreate.
	clean bool
}

// PlanUpload returns a mutator that registers the immutable snapshot as an internal
// resource. It builds the bundle zip, stages it under the bundle's local state
// directory as "<hash>.zip", and records the path on the resource; the staged file
// is uploaded when the resource is created on apply. Pass clean=true to first
// discard zips staged by a previous run; pass false when applying a pre-existing
// plan so its staged zip is preserved.
func PlanUpload(clean bool) bundle.Mutator {
	return &snapshotUpload{clean: clean}
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
	if _, ok := b.Config.Resources.Snapshots["immutable"]; ok {
		return nil
	}

	zipContent, fileCount, err := BundleZip(ctx, b)
	if err != nil {
		return diag.FromErr(fmt.Errorf("failed to build snapshot zip: %w", err))
	}

	// Discard any zip staged by a previous plan/deploy so the folder only ever holds
	// the current snapshot; content-addressed names would otherwise accumulate.
	if m.clean {
		if err := os.RemoveAll(b.GetLocalStateDir(ctx, "snapshots")); err != nil {
			return diag.FromErr(fmt.Errorf("failed to clean snapshot dir: %w", err))
		}
	}
	dir, err := b.LocalStateDir(ctx, "snapshots")
	if err != nil {
		return diag.FromErr(err)
	}
	zipPath := filepath.Join(dir, snapshot.HashFromContent(zipContent)+".zip")
	if err := os.WriteFile(zipPath, zipContent, 0o600); err != nil {
		return diag.FromErr(fmt.Errorf("failed to write snapshot zip: %w", err))
	}

	b.Config.Resources.Snapshots["immutable"] = &resources.Snapshot{
		BundleID:   BundleID(b),
		ACL:        BuildACL(b),
		RemoteRoot: remoteRoot,
		ZipPath:    filepath.ToSlash(zipPath),
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

// bundleIDNamespace is the UUID namespace used to derive the bundle ID.
var bundleIDNamespace = uuid.MustParse("4b4e4b5a-3c3d-4e4f-8b8c-9d9e9f0a0b0c")

// BundleID returns a stable UUID that identifies the bundle deployment.
// It is derived deterministically from workspace.state_path, which is the
// canonical unique identifier for a deployment (name, target, and workspace root
// are all encoded in it). Two bundles with the same name and target but different
// workspace.state_path values get distinct IDs.
func BundleID(b *bundle.Bundle) string {
	return uuid.NewSHA1(bundleIDNamespace, []byte(b.Config.Workspace.StatePath)).String()
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

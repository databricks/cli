package snapshot

import (
	"context"
	"fmt"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/bundle/direct/dresources"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/snapshot"
	"github.com/google/uuid"
)

// fileLimitWarning is the file count above which immutable folder deployments may fail.
const fileLimitWarning = 1000

type snapshotUpload struct {
	skipZip bool
}

// PlanUpload returns a mutator that registers the immutable snapshot as an internal
// resource. Unless skipZip is set, it also builds the bundle zip and stages it in
// memory on the resource; the zip is uploaded when the resource is created on apply.
func PlanUpload(skipZip bool) bundle.Mutator {
	return &snapshotUpload{skipZip: skipZip}
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
	if _, ok := b.Config.Resources.Snapshots["immutable"]; !ok {
		b.Config.Resources.Snapshots["immutable"] = &resources.Snapshot{
			BundleID:   BundleID(b),
			ACL:        BuildACL(b),
			RemoteRoot: remoteRoot,
		}
	}

	var diags diag.Diagnostics
	if !m.skipZip {
		zipContent, fileCount, err := BundleZip(ctx, b)
		if err != nil {
			return diag.FromErr(fmt.Errorf("failed to build snapshot zip: %w", err))
		}

		if fileCount > fileLimitWarning {
			diags = append(diags, diag.Warningf(
				"immutable folder deployment may not work correctly: bundle contains %d files (limit is %d)",
				fileCount, fileLimitWarning,
			)...)
		}

		b.Config.Resources.Snapshots["immutable"].ZipContent = string(zipContent)
	}

	return diags
}

// SyncZipContent copies the zip content from b.Config.Resources.Snapshots["immutable"]
// into the in-memory state cache entry for the snapshot resource. This is needed when
// deploying from a plan file: the plan JSON omits ZipContent (json:"-"), so InitForApply
// leaves it empty, causing DoCreate to upload an empty zip and derive a wrong snapshot ID.
func SyncZipContent(b *bundle.Bundle) {
	snap := b.Config.Resources.Snapshots["immutable"]
	if snap == nil || snap.ZipContent == "" {
		return
	}
	sv, ok := b.DeploymentBundle.StateCache.Load("resources.internal_immutable_snapshots.immutable")
	if !ok {
		return
	}
	state, ok := sv.Value.(*dresources.SnapshotState)
	if !ok {
		return
	}
	state.ZipContent = snap.ZipContent
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

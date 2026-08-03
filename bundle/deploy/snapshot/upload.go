package snapshot

import (
	"context"
	"fmt"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/snapshot"
)

// fileLimitWarning is the file count above which immutable folder deployments may fail.
const fileLimitWarning = 1000

type snapshotUpload struct {
	skipZip bool
	// uploader allows test injection of a custom SnapshotUploader.
	uploader snapshot.SnapshotUploader
}

// PlanUpload returns a mutator that builds the bundle zip, uploads it via
// /api/2.0/repos/snapshots, and registers the snapshot as an internal resource.
func PlanUpload(skipZip bool) bundle.Mutator {
	return &snapshotUpload{skipZip: skipZip}
}

func (m *snapshotUpload) Name() string {
	return "snapshot.Upload"
}

func (m *snapshotUpload) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	uploader := m.uploader
	if uploader == nil {
		var err error
		uploader, err = snapshot.NewSnapshotUploader(b.WorkspaceClient(ctx))
		if err != nil {
			return diag.FromErr(err)
		}
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
			BundleID:   b.DeploymentBundle.StateDB.GetOrInitLineage(),
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

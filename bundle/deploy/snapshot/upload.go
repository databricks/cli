package snapshot

import (
	"context"
	"fmt"
	"path"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/log"
)

// fileLimitWarning is the file count above which immutable folder deployments may fail.
const fileLimitWarning = 1000

type snapshotUpload struct {
	// uploader allows test injection of a custom SnapshotUploader.
	uploader SnapshotUploader
}

// Upload returns a mutator that builds the bundle zip, uploads it via
// /api/2.0/repos/snapshots, and updates workspace.file_path and
// workspace.artifact_path to the content-addressed location returned by the API.
func Upload() bundle.Mutator {
	return &snapshotUpload{}
}

func (m *snapshotUpload) Name() string {
	return "snapshot.Upload"
}

func (m *snapshotUpload) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	uploader := m.uploader
	if uploader == nil {
		var err error
		uploader, err = NewSnapshotUploader(b.WorkspaceClient(ctx))
		if err != nil {
			return diag.FromErr(err)
		}
	}

	cmdio.LogString(ctx, "Uploading immutable bundle snapshot...")

	zipContent, fileCount, err := BundleZip(ctx, b)
	if err != nil {
		return diag.FromErr(fmt.Errorf("failed to build snapshot zip: %w", err))
	}
	var diags diag.Diagnostics
	if fileCount > fileLimitWarning {
		diags = append(diags, diag.Warningf(
			"immutable folder deployment may not work correctly: bundle contains %d files (limit is %d)",
			fileCount, fileLimitWarning,
		)...)
	}
	snapshotID := IDFromContent(zipContent)
	log.Debugf(ctx, "snapshot.Upload: snapshotID=%s zip=%d bytes", snapshotID, len(zipContent))

	acl := BuildACL(b)
	// Use the deployment lineage UUID as bundle_id so the snapshot directory is
	// keyed to this specific deployment (not to the bundle name, which can be
	// reused across unrelated deployments).
	bundleID := b.DeploymentBundle.StateDB.GetOrInitLineage()
	info, err := uploader.Upload(ctx, bundleID, snapshotID, acl, zipContent)
	if err != nil {
		return diag.FromErr(err)
	}

	log.Infof(ctx, "Snapshot uploaded to %s", info.Path)

	// The API unpacks the zip under a "src" subdirectory.
	b.Config.Workspace.SnapshotPath = info.Path
	b.Config.Workspace.FilePath = path.Join(info.Path, "src", "files")
	// Only set artifact_path when artifacts are present; with no artifacts the
	// zip has no "src/artifacts" directory and a get-status on it would 404.
	if len(b.Config.Artifacts) > 0 {
		b.Config.Workspace.ArtifactPath = path.Join(info.Path, "src", "artifacts")
	}

	return diags
}

// BuildACL constructs the access_control_list for the snapshot upload.
// It grants CAN_READ to the current user and to every principal listed in the
// top-level permissions section of the bundle config.
func BuildACL(b *bundle.Bundle) []ACLEntry {
	acl := []ACLEntry{
		{UserName: b.Config.Workspace.CurrentUser.UserName, PermissionLevel: "CAN_READ"},
	}
	for _, p := range b.Config.Permissions {
		acl = append(acl, ACLEntry{
			UserName:             p.UserName,
			GroupName:            p.GroupName,
			ServicePrincipalName: p.ServicePrincipalName,
			PermissionLevel:      "CAN_READ",
		})
	}
	return acl
}

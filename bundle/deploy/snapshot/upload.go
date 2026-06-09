package snapshot

import (
	"context"
	"fmt"
	"path"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/filer"
	"github.com/databricks/cli/libs/log"
)

type snapshotUpload struct {
	// uploader allows test injection of a custom SnapshotUploader.
	uploader filer.SnapshotUploader
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
		uploader, err = filer.NewSnapshotUploader(b.WorkspaceClient(ctx))
		if err != nil {
			return diag.FromErr(err)
		}
	}

	cmdio.LogString(ctx, "Uploading immutable bundle snapshot...")

	zipContent, err := BundleZip(ctx, b)
	if err != nil {
		return diag.FromErr(fmt.Errorf("failed to build snapshot zip: %w", err))
	}
	snapshotID := IDFromContent(zipContent)
	log.Debugf(ctx, "snapshot.Upload: snapshotID=%s zip=%d bytes", snapshotID, len(zipContent))

	info, err := uploader.Upload(ctx, b.Config.Bundle.Name, snapshotID, b.Config.Workspace.CurrentUser.UserName, zipContent)
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

	return nil
}

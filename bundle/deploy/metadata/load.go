package metadata

import (
	"context"
	"encoding/json"
	"io"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/metadata"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/filer"
)

type load struct{}

// Load reads the metadata file written during the last deploy and populates
// fields on the bundle that are not available locally (e.g. workspace.snapshot_path
// for immutable bundles, which is only known after snapshot.Upload() ran).
func Load() bundle.Mutator {
	return &load{}
}

func (m *load) Name() string {
	return "metadata.Load"
}

func (m *load) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	f, err := filer.NewWorkspaceFilesClient(b.WorkspaceClient(ctx), b.Config.Workspace.StatePath)
	if err != nil {
		return diag.FromErr(err)
	}

	r, err := f.Read(ctx, metadataFileName)
	if err != nil {
		// Missing metadata file means the bundle was never deployed or was
		// deployed by an older CLI version that didn't write metadata. Treat
		// it as a no-op so destroy can still proceed.
		return nil
	}
	defer r.Close()

	raw, err := io.ReadAll(r)
	if err != nil {
		return diag.FromErr(err)
	}

	var md metadata.Metadata
	if err := json.Unmarshal(raw, &md); err != nil {
		return diag.FromErr(err)
	}

	if md.Config.Workspace.SnapshotPath != "" {
		b.Config.Workspace.SnapshotPath = md.Config.Workspace.SnapshotPath
	}

	return nil
}

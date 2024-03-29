package metadata

import (
	"bytes"
	"context"
	"encoding/json"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/filer"
)

const MetadataFileName = "metadata.json"

type upload struct{}

func Upload() bundle.Mutator {
	return &upload{}
}

func (m *upload) Name() string {
	return "metadata.Upload"
}

func (m *upload) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	f, err := filer.NewWorkspaceFilesClient(b.WorkspaceClient(), b.Config.Workspace.StatePath)
	if err != nil {
		return diag.FromErr(err)
	}

	metadata, err := json.MarshalIndent(b.Metadata, "", "  ")
	if err != nil {
		return diag.FromErr(err)
	}

	return diag.FromErr(f.Write(ctx, MetadataFileName, bytes.NewReader(metadata), filer.CreateParentDirectories, filer.OverwriteIfExists))
}

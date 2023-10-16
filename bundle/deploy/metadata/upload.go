package metadata

import (
	"bytes"
	"context"
	"encoding/json"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/filer"
)

var MetadataFileName = "metadata.json"

type upload struct{}

func Upload() bundle.Mutator {
	return &upload{}
}

func (m *upload) Name() string {
	return "metadata.Upload"
}

func (m *upload) Apply(ctx context.Context, b *bundle.Bundle) error {
	f, err := filer.NewWorkspaceFilesClient(b.WorkspaceClient(), b.Config.Workspace.StatePath)
	if err != nil {
		return err
	}

	metadata, err := json.MarshalIndent(b.Metadata, "", "  ")
	if err != nil {
		return err
	}

	return f.Write(ctx, MetadataFileName, bytes.NewReader(metadata), filer.CreateParentDirectories, filer.OverwriteIfExists)
}

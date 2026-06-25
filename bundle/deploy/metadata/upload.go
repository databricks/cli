package metadata

import (
	"bytes"
	"context"
	"encoding/json"
	"path"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/filer"
)

const metadataFileName = "metadata.json"

func metadataFilePath(b *bundle.Bundle) string {
	return path.Join(b.Config.Workspace.StatePath, metadataFileName)
}

type upload struct{}

func Upload() bundle.Mutator {
	return &upload{}
}

func (m *upload) Name() string {
	return "metadata.Upload"
}

func (m *upload) Apply(ctx context.Context, b *bundle.Bundle) error {
	f, err := filer.NewWorkspaceFilesClient(b.WorkspaceClient(ctx), b.Config.Workspace.StatePath)
	if err != nil {
		return err
	}

	metadata, err := json.MarshalIndent(b.Metadata, "", "  ")
	if err != nil {
		return err
	}

	return f.Write(ctx, metadataFileName, bytes.NewReader(metadata), filer.CreateParentDirectories, filer.OverwriteIfExists)
}

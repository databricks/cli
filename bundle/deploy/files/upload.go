package files

import (
	"context"
	"fmt"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/cmdio"
)

type upload struct{}

func (m *upload) Name() string {
	return "files.Upload"
}

func (m *upload) Apply(ctx context.Context, b *bundle.Bundle) ([]bundle.Mutator, error) {
	cmdio.LogString(ctx, "Starting upload of bundle files")
	sync, err := getSync(ctx, b)
	if err != nil {
		return nil, err
	}

	err = sync.RunOnce(ctx)
	if err != nil {
		return nil, err
	}

	cmdio.LogString(ctx, fmt.Sprintf("Uploaded bundle files at %s!\n", b.Config.Workspace.FilesPath))
	return nil, nil
}

func Upload() bundle.Mutator {
	return &upload{}
}

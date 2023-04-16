package files

import (
	"context"

	"github.com/databricks/bricks/bundle"
	"github.com/databricks/bricks/libs/cmdio"
)

type upload struct{}

func (m *upload) Name() string {
	return "files.Upload"
}

func (m *upload) Apply(ctx context.Context, b *bundle.Bundle) ([]bundle.Mutator, error) {
	cmdio.Log(ctx, NewUploadStartedEvent())
	sync, err := getSync(ctx, b)
	if err != nil {
		cmdio.Log(ctx, NewDeleteFailedEvent())
		return nil, err
	}

	err = sync.RunOnce(ctx)
	if err != nil {
		cmdio.Log(ctx, NewDeleteFailedEvent())
		return nil, err
	}

	cmdio.Log(ctx, NewUploadCompletedEvent(b.Config.Workspace.FilesPath))
	return nil, nil
}

func Upload() bundle.Mutator {
	return &upload{}
}

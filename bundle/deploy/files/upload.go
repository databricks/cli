package files

import (
	"context"

	"github.com/databricks/bricks/bundle"
)

type upload struct{}

func (m *upload) Name() string {
	return "files.Upload"
}

func (m *upload) Apply(ctx context.Context, b *bundle.Bundle) ([]bundle.Mutator, error) {
	sync, err := getSync(ctx, b)
	if err != nil {
		return nil, err
	}

	err = sync.RunOnce(ctx)
	if err != nil {
		return nil, err
	}

	return nil, nil
}

func Upload() bundle.Mutator {
	return &upload{}
}

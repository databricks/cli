package whl

import (
	"context"
	"fmt"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/python"
)

type infer struct {
	name string
}

func (m *infer) Apply(ctx context.Context, b *bundle.Bundle) error {
	artifact := b.Config.Artifacts[m.name]
	py, err := python.DetectExecutable(ctx)
	if err != nil {
		return err
	}
	artifact.BuildCommand = fmt.Sprintf("%s -m pip install setuptools wheel && %s setup.py bdist_wheel", py, py)

	return nil
}

func (m *infer) Name() string {
	return fmt.Sprintf("artifacts.whl.Infer(%s)", m.name)
}

func InferBuildCommand(name string) bundle.Mutator {
	return &infer{
		name: name,
	}
}

package wheel

import (
	"context"
	"fmt"
	"os"
	"path"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/python"
)

type build struct {
	name string
}

func Build(name string) bundle.Mutator {
	return &build{
		name: name,
	}
}

func (m *build) Name() string {
	return fmt.Sprintf("wheel.Build(%s)", m.name)
}

func (m *build) Apply(ctx context.Context, b *bundle.Bundle) ([]bundle.Mutator, error) {
	a, ok := b.Config.Artifacts[m.name]
	if !ok {
		return nil, fmt.Errorf("artifact doesn't exist: %s", m.name)
	}
	cmdio.LogString(ctx, fmt.Sprintf("Starting build of Python wheel artifact: %s", m.name))

	artifact := a.PythonPackage
	if artifact.Path == "" {
		d, _ := os.Getwd()
		artifact.Path = d
	}
	if _, err := os.Stat(artifact.Path); os.IsNotExist(err) {
		return nil, fmt.Errorf("artifact path does't exists: %s", artifact.Path)
	}
	wheelPath, err := python.BuildWheel(ctx, artifact.Path)
	if err != nil {
		return nil, err
	}

	artifact.LocalPath = wheelPath

	remotePath := b.Config.Workspace.ArtifactsPath
	if remotePath == "" {
		return nil, fmt.Errorf("remote artifact path not configured")
	}
	artifact.RemotePath = path.Join(remotePath, path.Base(wheelPath))
	cmdio.LogString(ctx, fmt.Sprintf("Built Python wheel artifact: %s", m.name))

	return nil, nil
}

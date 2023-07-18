package whl

import (
	"context"
	"fmt"
	"os"
	"path"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs"
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
	return fmt.Sprintf("artifacts.whl.Build(%s)", m.name)
}

func (m *build) Apply(ctx context.Context, b *bundle.Bundle) error {
	artifact, ok := b.Config.Artifacts[m.name]
	if !ok {
		return fmt.Errorf("artifact doesn't exist: %s", m.name)
	}

	if artifact.BuildCommand == "" {
		//TODO: infer build step if not provided
		return fmt.Errorf("artifacts.whl.Build(%s): missing build property for the artifact", m.name)
	}

	cmdio.LogString(ctx, fmt.Sprintf("artifacts.whl.Build(%s): Building...", m.name))

	dir := artifact.Path

	defer libs.ChdirAndBack(dir)
	os.RemoveAll("dist")
	python.CleanupWheelFolder(".")

	out, err := artifact.Build(ctx)
	if err != nil {
		return fmt.Errorf("artifacts.whl.Build(%s): Failed %w, output: %s", m.name, err, out)
	}
	cmdio.LogString(ctx, fmt.Sprintf("artifacts.whl.Build(%s): Build succeeded", m.name))

	wheel := python.FindFileWithSuffixInPath("dist", ".whl")
	if wheel == "" {
		return fmt.Errorf("artifacts.whl.Build(%s): cannot find built wheel in %s", m.name, dir)
	}
	artifact.File = path.Join(dir, wheel)

	return nil
}

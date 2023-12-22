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

	// Note: using --build-number (build tag) flag does not help with re-installing
	// libraries on all-purpose clusters. The reason is that `pip` ignoring build tag
	// when upgrading the library and only look at wheel version.
	// Build tag is only used for sorting the versions and the one with higher build tag takes priority when installed.
	// It only works if no library is installed
	// See https://github.com/pypa/pip/blob/a15dd75d98884c94a77d349b800c7c755d8c34e4/src/pip/_internal/index/package_finder.py#L522-L556
	// https://github.com/pypa/pip/issues/4781
	//
	// Thus, the only way to reinstall the library on all-purpose cluster is to increase wheel version manually or
	// use automatic version generation, f.e.
	// setup(
	//   version=datetime.datetime.utcnow().strftime("%Y%m%d.%H%M%S"),
	// ...
	//)
	artifact.BuildCommand = fmt.Sprintf(`"%s" setup.py bdist_wheel`, py)

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

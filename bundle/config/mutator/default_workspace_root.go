package mutator

import (
	"context"
	"fmt"
	"path"
	"strings"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/diag"
)

// How the bundle name and target appear in a configured root_path, which this mutator
// sees before variable resolution replaces them.
const (
	bundleNameRef   = "${bundle.name}"
	bundleTargetRef = "${bundle.target}"
)

type defineDefaultWorkspaceRoot struct{}

// DefineDefaultWorkspaceRoot defines the default workspace root path.
func DefineDefaultWorkspaceRoot() bundle.Mutator {
	return &defineDefaultWorkspaceRoot{}
}

func (m *defineDefaultWorkspaceRoot) Name() string {
	return "DefineDefaultWorkspaceRoot"
}

func (m *defineDefaultWorkspaceRoot) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	if b.Config.Workspace.RootPath != "" {
		b.RootPathIsNameTargetScoped = endsWithNameAndTarget(b.Config.Workspace.RootPath)
		return nil
	}

	if b.Config.Bundle.Name == "" {
		return diag.Errorf("unable to define default workspace root: bundle name not defined")
	}

	if b.Config.Bundle.Target == "" {
		return diag.Errorf("unable to define default workspace root: bundle target not selected")
	}

	b.Config.Workspace.RootPath = fmt.Sprintf(
		"~/.bundle/%s/%s",
		b.Config.Bundle.Name,
		b.Config.Bundle.Target,
	)
	b.RootPathIsNameTargetScoped = true
	return nil
}

// endsWithNameAndTarget reports whether the last two segments of rootPath are the
// bundle name and target references.
func endsWithNameAndTarget(rootPath string) bool {
	rootPath = strings.TrimSuffix(rootPath, "/")
	return path.Base(rootPath) == bundleTargetRef && path.Base(path.Dir(rootPath)) == bundleNameRef
}

package mutator

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/databricks/cli/bundle"

	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/structs/structpath"
	"github.com/databricks/cli/libs/structs/structvar"
)

type rewriteSyncPaths struct{}

func RewriteSyncPaths() bundle.Mutator {
	return &rewriteSyncPaths{}
}

func (m *rewriteSyncPaths) Name() string {
	return "RewriteSyncPaths"
}

// makeRelativeTo joins the relative path of the file the string node was
// defined in w.r.t. the bundle root path, with the contents of the string node.
//
// For example:
//   - The bundle root is /foo
//   - The configuration file that defines the string node is at /foo/bar/baz.yml
//   - The string node contains "somefile.*"
//
// Then the resulting value will be "bar/somefile.*".
func (m *rewriteSyncPaths) makeRelativeTo(root string, v structvar.View) (string, error) {
	dir := filepath.Dir(v.Location().File)
	rel, err := filepath.Rel(root, dir)
	if err != nil {
		return "", err
	}

	s, ok := v.AsString()
	if !ok {
		return "", fmt.Errorf("expected string value but got %s", v.Kind())
	}

	return filepath.Join(rel, s), nil
}

func (m *rewriteSyncPaths) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	rewrite := func(field string, toSlash bool) error {
		pattern := structpath.NewPattern(nil, "sync", field, structpath.AnyIndex)
		return structvar.ForEach(b.Config.View(), pattern, func(p *structpath.PathNode, v structvar.View) error {
			path, err := m.makeRelativeTo(b.BundleRootPath, v)
			if err != nil {
				return err
			}
			if toSlash {
				path = filepath.ToSlash(path)
			}
			return b.Config.Set(p, path)
		})
	}

	// Makes include and exclude paths relative to the bundle root first.
	// Then converts them to use Unix-style slashes.
	// This is required for the ignore.GitIgnore we use in libs/fileset to work correctly.
	err := rewrite("paths", false)
	if err == nil {
		err = rewrite("include", true)
	}
	if err == nil {
		err = rewrite("exclude", true)
	}

	return diag.FromErr(err)
}

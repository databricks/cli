package mutator

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/databricks/cli/bundle"

	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/dyn"
)

type rewriteSyncPaths struct{}

func RewriteSyncPaths() bundle.Mutator {
	return &rewriteSyncPaths{}
}

func (m *rewriteSyncPaths) Name() string {
	return "RewriteSyncPaths"
}

// makeRelativeTo returns a dyn.MapFunc that joins the relative path
// of the file it was defined in w.r.t. the bundle root path, with
// the contents of the string node.
//
// For example:
//   - The bundle root is /foo
//   - The configuration file that defines the string node is at /foo/bar/baz.yml
//   - The string node contains "somefile.*"
//
// Then the resulting value will be "bar/somefile.*".
func (m *rewriteSyncPaths) makeRelativeTo(root string) dyn.MapFunc {
	return func(_ dyn.Path, v dyn.Value) (dyn.Value, error) {
		dir := filepath.Dir(v.Location().File)
		rel, err := filepath.Rel(root, dir)
		if err != nil {
			return dyn.InvalidValue, err
		}

		return dyn.NewValue(filepath.Join(rel, v.MustString()), v.Locations()), nil
	}
}

func (m *rewriteSyncPaths) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	err := b.Config.Mutate(func(v dyn.Value) (dyn.Value, error) {
		return dyn.Map(v, "sync", func(_ dyn.Path, v dyn.Value) (nv dyn.Value, err error) {
			v, err = dyn.Map(v, "paths", dyn.Foreach(m.makeRelativeTo(b.BundleRootPath)))
			if err != nil {
				return dyn.InvalidValue, err
			}

			makeRelativeFn := m.makeRelativeTo(b.BundleRootPath)

			// Makes include and exclude paths relative to the bundle root first.
			// Then converts them to use Unix-style slashes.
			// This is required for the ignore.GitIgnore we use in libs/fileset to work correctly.
			v, err = dyn.Map(v, "include", dyn.Foreach(func(p dyn.Path, val dyn.Value) (dyn.Value, error) {
				relPath, err := makeRelativeFn(p, val)
				if err != nil {
					return dyn.InvalidValue, err
				}
				str, ok := relPath.AsString()
				if !ok {
					return dyn.InvalidValue, fmt.Errorf("expected string value but got %s", relPath.Kind())
				}
				return dyn.NewValue(filepath.ToSlash(str), relPath.Locations()), nil
			}))
			if err != nil {
				return dyn.InvalidValue, err
			}

			v, err = dyn.Map(v, "exclude", dyn.Foreach(func(p dyn.Path, val dyn.Value) (dyn.Value, error) {
				relPath, err := makeRelativeFn(p, val)
				if err != nil {
					return dyn.InvalidValue, err
				}
				str, ok := relPath.AsString()
				if !ok {
					return dyn.InvalidValue, fmt.Errorf("expected string value but got %s", relPath.Kind())
				}
				return dyn.NewValue(filepath.ToSlash(str), relPath.Locations()), nil
			}))
			if err != nil {
				return dyn.InvalidValue, err
			}
			return v, nil
		})
	})

	return diag.FromErr(err)
}

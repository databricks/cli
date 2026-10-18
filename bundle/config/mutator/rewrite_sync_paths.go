package mutator

import (
	"context"
	"path/filepath"

	"github.com/databricks/cli/bundle"

	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/structs/structpath"
)

type rewriteSyncPaths struct{}

func RewriteSyncPaths() bundle.Mutator {
	return &rewriteSyncPaths{}
}

func (m *rewriteSyncPaths) Name() string {
	return "RewriteSyncPaths"
}

// makeRelativeTo joins the relative path of the file the value was defined in
// (loc) w.r.t. the bundle root path, with the value s.
//
// For example:
//   - The bundle root is /foo
//   - The configuration file that defines the value is at /foo/bar/baz.yml
//   - The value is "somefile.*"
//
// Then the resulting value will be "bar/somefile.*".
func (m *rewriteSyncPaths) makeRelativeTo(root string, loc diag.Location, s string) (string, error) {
	rel, err := filepath.Rel(root, filepath.Dir(loc.File))
	if err != nil {
		return "", err
	}
	return filepath.Join(rel, s), nil
}

// rewrite rewrites the sync paths in values (the field key of sync) relative to the
// bundle root, with Unix-style slashes if toSlash.
func (m *rewriteSyncPaths) rewrite(b *bundle.Bundle, key string, values []string, toSlash bool) error {
	for i := range values {
		loc := b.Config.GetLocationOfExact(structpath.NewPath(nil, "sync", key, i))
		path, err := m.makeRelativeTo(b.BundleRootPath, loc, values[i])
		if err != nil {
			return err
		}
		if toSlash {
			path = filepath.ToSlash(path)
		}
		values[i] = path
	}
	return nil
}

func (m *rewriteSyncPaths) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	// Makes include and exclude paths relative to the bundle root first.
	// Then converts them to use Unix-style slashes.
	// This is required for the ignore.GitIgnore we use in libs/fileset to work correctly.
	sync := &b.Config.Sync
	err := m.rewrite(b, "paths", sync.Paths, false)
	if err == nil {
		err = m.rewrite(b, "include", sync.Include, true)
	}
	if err == nil {
		err = m.rewrite(b, "exclude", sync.Exclude, true)
	}

	return diag.FromErr(err)
}

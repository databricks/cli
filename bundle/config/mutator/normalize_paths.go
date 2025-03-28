package mutator

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config/mutator/paths"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/dyn"
)

type normalizePaths struct{}

func (a normalizePaths) Name() string {
	return "NormalizePaths"
}

func (a normalizePaths) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	err := b.Config.Mutate(func(configValue dyn.Value) (dyn.Value, error) {
		return paths.VisitJobPaths(configValue, func(path dyn.Path, kind paths.PathKind, v dyn.Value) (dyn.Value, error) {
			value := v.MustString()

			if filepath.IsAbs(value) {
				return v.WithDirectory(b.BundleRootPath), nil
			}

			absoluteDir, err := v.Directory()
			if err != nil {
				return dyn.InvalidValue, fmt.Errorf("unable to determine directory for a value at %s: %w", path.String(), err)
			}

			relDir, err := filepath.Rel(b.BundleRootPath, absoluteDir)
			if err != nil {
				return dyn.InvalidValue, err
			}

			return v.WithValue(filepath.Join(relDir, value)).WithDirectory(b.BundleRootPath), nil
		})
	})
	if err != nil {
		return diag.FromErr(fmt.Errorf("failed to relativize paths against bundle root: %w", err))
	}

	return diag.FromErr(err)
}

func NormalizePaths() bundle.Mutator {
	return &normalizePaths{}
}

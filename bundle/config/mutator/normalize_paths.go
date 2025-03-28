package mutator

import (
	"context"
	"fmt"
	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config/mutator/paths"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/dyn"
	"path/filepath"
)

type normalizePaths struct{}

func (a normalizePaths) Name() string {
	return "NormalizePaths"
}

func (a normalizePaths) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	err := b.Config.Mutate(func(configValue dyn.Value) (dyn.Value, error) {
		return paths.VisitJobPaths(configValue, func(path dyn.Path, kind paths.PathKind, value dyn.Value) (dyn.Value, error) {
			dir, err := value.Location().Directory()
			if err != nil {
				return dyn.InvalidValue, fmt.Errorf("unable to determine directory for a value at %s: %w", path.String(), err)
			}

			newPath := filepath.Join(dir, value.MustString())
			newValue := dyn.NewValue(newPath, value.Locations())

			return newValue, nil

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

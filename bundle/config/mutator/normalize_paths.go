package mutator

import (
	"context"
	"errors"
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

func NormalizePaths() bundle.Mutator {
	return &normalizePaths{}
}

func (a normalizePaths) Apply(_ context.Context, b *bundle.Bundle) diag.Diagnostics {
	normalizeFn := func(path dyn.Path, kind paths.TranslateMode, v dyn.Value) (dyn.Value, error) {
		value := v.MustString()

		//
		if filepath.IsAbs(value) {
			return v, nil
		}

		dir, err := locationDirectory(v.Location())
		if err != nil {
			return dyn.InvalidValue, fmt.Errorf("unable to determine directory for a value at %s: %w", path.String(), err)
		}

		relDir, err := filepath.Rel(b.BundleRootPath, dir)
		if err != nil {
			return dyn.InvalidValue, err
		}

		return dyn.NewValue(filepath.Join(relDir, value), v.Locations()), nil
	}

	err := b.Config.Mutate(func(v0 dyn.Value) (dyn.Value, error) {
		visitors := []func(v0 dyn.Value, fn paths.VisitFunc) (dyn.Value, error){
			paths.VisitJobPaths,
			paths.VisitAppPaths,
			paths.VisitArtifactPaths,
			paths.VisitDashboardPaths,
			paths.VisitPipelinePaths,
		}

		v := v0
		for _, visitor := range visitors {
			v1, err := visitor(v, normalizeFn)
			if err != nil {
				return dyn.InvalidValue, fmt.Errorf("failed to normalize paths: %w", err)
			}
			v = v1
		}

		return v, nil
	})
	if err != nil {
		return diag.FromErr(fmt.Errorf("failed to normalize job paths: %w", err))
	}

	return diag.FromErr(err)
}

func locationDirectory(l dyn.Location) (string, error) {
	if l.File == "" {
		return "", errors.New("no file in location")
	}

	return filepath.Dir(l.File), nil
}

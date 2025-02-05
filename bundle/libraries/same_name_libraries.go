package libraries

import (
	"context"
	"path/filepath"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/dyn"
)

type checkForSameNameLibraries struct{}

var patterns = []dyn.Pattern{
	taskLibrariesPattern.Append(dyn.AnyIndex(), dyn.AnyKey()),
	forEachTaskLibrariesPattern.Append(dyn.AnyIndex(), dyn.AnyKey()),
	envDepsPattern.Append(dyn.AnyIndex()),
}

func (c checkForSameNameLibraries) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	var diags diag.Diagnostics
	libBaseNames := make(map[string]bool)

	err := b.Config.Mutate(func(v dyn.Value) (dyn.Value, error) {
		var err error
		for _, pattern := range patterns {
			v, err = dyn.MapByPattern(v, pattern, func(p dyn.Path, lv dyn.Value) (dyn.Value, error) {
				libPath := lv.MustString()
				// If not local library, skip the check
				if !IsLibraryLocal(libPath) {
					return lv, nil
				}

				lib := filepath.Base(lv.MustString())
				if libBaseNames[lib] {
					diags = append(diags, diag.Diagnostic{
						Severity:  diag.Error,
						Summary:   "Duplicate local library name",
						Detail:    "Local library names must be unique",
						Locations: lv.Locations(),
						Paths:     []dyn.Path{p},
					})
				}

				libBaseNames[lib] = true
				return lv, nil
			})
			if err != nil {
				return dyn.InvalidValue, err
			}
		}

		if err != nil {
			return dyn.InvalidValue, err
		}

		return v, nil
	})
	if err != nil {
		diags = diags.Extend(diag.FromErr(err))
	}

	return diags
}

func (c checkForSameNameLibraries) Name() string {
	return "CheckForSameNameLibraries"
}

func CheckForSameNameLibraries() bundle.Mutator {
	return checkForSameNameLibraries{}
}

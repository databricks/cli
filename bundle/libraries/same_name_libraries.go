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

type libData struct {
	fullPath  string
	locations []dyn.Location
	paths     []dyn.Path
}

func (c checkForSameNameLibraries) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	var diags diag.Diagnostics
	libs := make(map[string]*libData)

	err := b.Config.Mutate(func(v dyn.Value) (dyn.Value, error) {
		var err error
		for _, pattern := range patterns {
			v, err = dyn.MapByPattern(v, pattern, func(p dyn.Path, lv dyn.Value) (dyn.Value, error) {
				libPath, ok := lv.AsString()
				if !ok {
					return lv, nil
				}

				// If not local library, skip the check
				if !IsLibraryLocal(libPath) {
					return lv, nil
				}

				libFullPath := lv.MustString()
				lib := filepath.Base(libFullPath)
				// If the same basename was seen already but full path is different
				// then it's a duplicate. Add the location to the location list.
				lp, ok := libs[lib]
				if !ok {
					libs[lib] = &libData{
						fullPath:  libFullPath,
						locations: []dyn.Location{lv.Location()},
						paths:     []dyn.Path{p},
					}
				} else if lp.fullPath != libFullPath {
					lp.locations = append(lp.locations, lv.Location())
					lp.paths = append(lp.paths, p)
				}

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

	// Iterate over all the libraries and check if there are any duplicates.
	// Duplicates will have more than one location.
	// If there are duplicates, add a diagnostic.
	for lib, lv := range libs {
		if len(lv.locations) > 1 {
			diags = append(diags, diag.Diagnostic{
				Severity:  diag.Error,
				Summary:   "Duplicate local library name " + lib,
				Detail:    "Local library names must be unique",
				Locations: lv.locations,
				Paths:     lv.paths,
			})
		}
	}
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

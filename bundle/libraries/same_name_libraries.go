package libraries

import (
	"context"
	"path/filepath"
	"strings"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/dyn"
)

type checkForSameNameLibraries struct{}

var patterns = []dyn.Pattern{
	taskLibrariesPattern.Append(dyn.AnyIndex(), dyn.Key("whl")),
	taskLibrariesPattern.Append(dyn.AnyIndex(), dyn.Key("jar")),
	forEachTaskLibrariesPattern.Append(dyn.AnyIndex(), dyn.Key("whl")),
	forEachTaskLibrariesPattern.Append(dyn.AnyIndex(), dyn.Key("jar")),
	envDepsPattern.Append(dyn.AnyIndex()),
	pipelineEnvDepsPattern.Append(dyn.AnyIndex()),
}

type libData struct {
	fullPath   string
	locations  []dyn.Location
	paths      []dyn.Path
	otherPaths []string
}

func (c checkForSameNameLibraries) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	var diags diag.Diagnostics
	libs := make(map[string]*libData)

	err := b.Config.Mutate(func(rootConfig dyn.Value) (dyn.Value, error) {
		var err error
		for _, pattern := range patterns {
			rootConfig, err = dyn.MapByPattern(rootConfig, pattern, func(p dyn.Path, libraryValue dyn.Value) (dyn.Value, error) {
				libPath, ok := libraryValue.AsString()
				if !ok {
					return libraryValue, nil
				}

				// If not local library, skip the check
				if !IsLibraryLocal(libPath) {
					return libraryValue, nil
				}

				lib := filepath.Base(libPath)
				// If the same basename was seen already but full path is different
				// then it's a duplicate. Add the location to the location list.
				lp, ok := libs[lib]
				if !ok {
					libs[lib] = &libData{
						fullPath:   libPath,
						locations:  []dyn.Location{libraryValue.Location()},
						paths:      []dyn.Path{p},
						otherPaths: []string{},
					}
				} else if lp.fullPath != libPath {
					lp.locations = append(lp.locations, libraryValue.Location())
					lp.paths = append(lp.paths, p)
					lp.otherPaths = append(lp.otherPaths, libPath)
				}

				return libraryValue, nil
			})
			if err != nil {
				return dyn.InvalidValue, err
			}
		}

		if err != nil {
			return dyn.InvalidValue, err
		}

		return rootConfig, nil
	})

	// Iterate over all the libraries and check if there are any duplicates.
	// Duplicates will have more than one location.
	// If there are duplicates, add a diagnostic.
	for lib, lv := range libs {
		if len(lv.locations) > 1 {
			diags = append(diags, diag.Diagnostic{
				Severity:  diag.Error,
				Summary:   "Duplicate local library names: " + lib,
				Detail:    "Local library names must be unique but found libraries with the same name: " + lv.fullPath + ", " + strings.Join(lv.otherPaths, ", "),
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

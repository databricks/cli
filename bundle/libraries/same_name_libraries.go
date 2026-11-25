package libraries

import (
	"context"
	"path/filepath"
	"strings"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/structs/structpath"
)

type checkForSameNameLibraries struct{}

// sameNamePatterns returns the patterns of the library values in the
// locations that hold lists, in the order of [locationsByKind]. Of two
// libraries with the same name, the one visited first is reported first.
func sameNamePatterns() []dyn.Pattern {
	var out []dyn.Pattern
	for _, l := range locationsByKind() {
		switch l.Kind {
		case resources.LibraryList, resources.DependencyList:
			out = append(out, l.valuePatterns()...)
		case resources.LibraryFile:
			// Not checked; see [resources.LibraryFile].
		}
	}
	return out
}

type libData struct {
	fullPath   string
	locations  []diag.Location
	paths      []*structpath.PathNode
	otherPaths []string
}

func (c checkForSameNameLibraries) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	var diags diag.Diagnostics
	libs := make(map[string]*libData)

	err := b.Config.Mutate(func(rootConfig dyn.Value) (dyn.Value, error) {
		var err error
		for _, pattern := range sameNamePatterns() {
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
						locations:  []diag.Location{libraryValue.Location()},
						paths:      dyn.ToStructPaths(p),
						otherPaths: []string{},
					}
				} else if lp.fullPath != libPath {
					lp.locations = append(lp.locations, libraryValue.Location())
					lp.paths = append(lp.paths, dyn.ToStructPath(p))
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

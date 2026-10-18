package libraries

import (
	"context"
	"path/filepath"
	"strings"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/structs/structpath"
)

type checkForSameNameLibraries struct{}


type libData struct {
	fullPath   string
	locations  []diag.Location
	paths      []*structpath.PathNode
	otherPaths []string
}

func (c checkForSameNameLibraries) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	var diags diag.Diagnostics
	libs := make(map[string]*libData)

	err := visitLibraryValues(&b.Config.Resources, false, func(p *structpath.PathNode, s *string) error {
		libPath := *s

		// If not local library, skip the check
		if !IsLibraryLocal(libPath) {
			return nil
		}

		lib := filepath.Base(libPath)
		loc := b.Config.GetLocationOfExact(p)
		// If the same basename was seen already but full path is different
		// then it's a duplicate. Add the location to the location list.
		lp, ok := libs[lib]
		if !ok {
			libs[lib] = &libData{
				fullPath:   libPath,
				locations:  []diag.Location{loc},
				paths:      []*structpath.PathNode{p},
				otherPaths: []string{},
			}
		} else if lp.fullPath != libPath {
			lp.locations = append(lp.locations, loc)
			lp.paths = append(lp.paths, p)
			lp.otherPaths = append(lp.otherPaths, libPath)
		}

		return nil
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

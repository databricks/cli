package libraries

import (
	"context"
	"path/filepath"
	"strings"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/structs/structpath"
	"github.com/databricks/cli/libs/structs/structvar"
)

type checkForSameNameLibraries struct{}

var patterns = []*structpath.PatternNode{
	structpath.MustParsePattern(taskLibrariesPattern.String() + "[*].whl"),
	structpath.MustParsePattern(taskLibrariesPattern.String() + "[*].jar"),
	structpath.MustParsePattern(forEachTaskLibrariesPattern.String() + "[*].whl"),
	structpath.MustParsePattern(forEachTaskLibrariesPattern.String() + "[*].jar"),
	structpath.MustParsePattern(clusterLibrariesPattern.String() + "[*].whl"),
	structpath.MustParsePattern(clusterLibrariesPattern.String() + "[*].jar"),
	structpath.MustParsePattern(envDepsPattern.String() + "[*]"),
	structpath.MustParsePattern(pipelineEnvDepsPattern.String() + "[*]"),
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

	root := b.Config.View()
	var err error
	for _, pattern := range patterns {
		err = structvar.ForEach(root, pattern, func(p *structpath.PathNode, libraryValue structvar.View) error {
			libPath, ok := libraryValue.AsString()
			if !ok {
				return nil
			}

			// If not local library, skip the check
			if !IsLibraryLocal(libPath) {
				return nil
			}

			lib := filepath.Base(libPath)
			// If the same basename was seen already but full path is different
			// then it's a duplicate. Add the location to the location list.
			lp, ok := libs[lib]
			if !ok {
				libs[lib] = &libData{
					fullPath:   libPath,
					locations:  []diag.Location{libraryValue.Location()},
					paths:      []*structpath.PathNode{p},
					otherPaths: []string{},
				}
			} else if lp.fullPath != libPath {
				lp.locations = append(lp.locations, libraryValue.Location())
				lp.paths = append(lp.paths, p)
				lp.otherPaths = append(lp.otherPaths, libPath)
			}

			return nil
		})
		if err != nil {
			break
		}
	}

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

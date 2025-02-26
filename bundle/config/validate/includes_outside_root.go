package validate

import (
	"context"
	"path/filepath"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/dyn"
)

type includesOutsideRoot struct{}

// Defining an include section outside databricks.yml is a no-op and users should
// be warned about it.
func IncludesOutsideRoot() bundle.ReadOnlyMutator {
	return &includesOutsideRoot{}
}

func (m *includesOutsideRoot) Name() string {
	return "validate:IncludesOutsideRoot"
}

func (m *includesOutsideRoot) Apply(ctx context.Context, rb bundle.ReadOnlyBundle) diag.Diagnostics {
	includesV := rb.Config().Value().Get("include")

	// return early if no includes are defined or the includes block is empty.
	if includesV.Kind() == dyn.KindInvalid || includesV.Kind() == dyn.KindNil {
		return nil
	}

	badLocations := []dyn.Location{}

	_, err := dyn.MapByPattern(
		rb.Config().Value().Get("include"),
		dyn.NewPattern(dyn.AnyIndex()),
		func(p dyn.Path, v dyn.Value) (dyn.Value, error) {
			for _, loc := range v.Locations() {
				isRoot := false
				for _, rootFileName := range config.FileNames {
					if loc.File == filepath.Join(rb.RootPath(), rootFileName) {
						isRoot = true
						break
					}
				}

				if !isRoot {
					badLocations = append(badLocations, loc)
				}
			}

			return v, nil
		},
	)
	if err != nil {
		return diag.FromErr(err)
	}

	if len(badLocations) == 0 {
		return nil
	}

	return diag.Diagnostics{
		{
			Severity: diag.Warning,
			Summary:  "Include section is defined outside root file",
			Detail: `The include section is defined in a file that is not the root file.
These values will be ignored because only the includes defined in
the bundle root file (that is databricks.yml or databricks.yaml)
are loaded.`,
			Locations: badLocations,
			Paths:     []dyn.Path{dyn.MustPathFromString("include")},
		},
	}
}

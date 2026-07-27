package validate

import (
	"cmp"
	"fmt"
	"slices"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/dyn"
)

// pathDiags collects errors attached to configuration paths. Validators that
// iterate resource maps use it so their output is ordered by path rather than by
// Go's randomized map iteration.
type pathDiags struct {
	b     *bundle.Bundle
	diags diag.Diagnostics
}

// errorf reports an error about the value at path, located wherever the config
// defines it.
func (p *pathDiags) errorf(path, format string, args ...any) {
	p.errorAt(dyn.MustPathFromString(path), p.b.Config.GetLocations(path), format, args...)
}

// errorAt reports an error about path at explicit locations. Use it when the
// interesting location is not where path is defined, e.g. the reference that
// points at it.
func (p *pathDiags) errorAt(path dyn.Path, locations []dyn.Location, format string, args ...any) {
	p.diags = append(p.diags, diag.Diagnostic{
		Severity:  diag.Error,
		Summary:   fmt.Sprintf(format, args...),
		Paths:     []dyn.Path{path},
		Locations: locations,
	})
}

func (p *pathDiags) sorted() diag.Diagnostics {
	slices.SortFunc(p.diags, func(x, y diag.Diagnostic) int {
		return cmp.Compare(x.Paths[0].String(), y.Paths[0].String())
	})
	return p.diags
}

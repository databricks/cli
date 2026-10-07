package validate

import (
	"context"
	"fmt"
	"maps"
	"reflect"
	"slices"
	"strings"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/structs/structpath"
)

func AllResourcesHaveValues() bundle.Mutator {
	return &allResourcesHaveValues{}
}

type allResourcesHaveValues struct{}

func (m *allResourcesHaveValues) Name() string {
	return "validate:AllResourcesHaveValues"
}

func (m *allResourcesHaveValues) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	diags := diag.Diagnostics{}

	for _, group := range b.Config.Resources.AllResources() {
		for _, rName := range slices.Sorted(maps.Keys(group.Resources)) {
			// A resource declared without a body is a nil pointer.
			if !reflect.ValueOf(group.Resources[rName]).IsNil() {
				continue
			}

			// Type of the resource, stripped of the trailing 's' to make it
			// singular.
			rType := strings.TrimSuffix(group.Description.PluralName, "s")

			p := structpath.NewStringKeys(nil, "resources", group.Description.PluralName, rName)
			diags = append(diags, diag.Diagnostic{
				Severity:  diag.Error,
				Summary:   fmt.Sprintf("%s %s is not defined", rType, rName),
				Locations: b.Config.LocationsAt(p),
				Paths:     []*structpath.PathNode{p},
			})
		}
	}

	return diags
}

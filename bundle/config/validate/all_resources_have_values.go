package validate

import (
	"context"
	"fmt"
	"strings"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
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

	b.Config.Resources.IterateAllResources(func(typ, rName string, res config.ConfigResource) bool {
		// A resource declared without a body is visited with a nil res.
		if res != nil {
			return true
		}

		// Type of the resource, stripped of the trailing 's' to make it
		// singular.
		rType := strings.TrimSuffix(typ, "s")

		p := structpath.NewPath(nil, "resources", typ, rName)
		diags = append(diags, diag.Diagnostic{
			Severity:  diag.Error,
			Summary:   fmt.Sprintf("%s %s is not defined", rType, rName),
			Locations: b.Config.GetLocationsOfExact(p),
			Paths:     []*structpath.PathNode{p},
		})
		return true
	})

	return diags
}

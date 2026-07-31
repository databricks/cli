package mutator

import (
	"context"
	"reflect"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/structs/structpath"
	"github.com/databricks/cli/libs/structs/structtag"
	"github.com/databricks/cli/libs/structs/structwalk"
)

type rejectInternalResources struct{}

// RejectInternalResources returns a mutator that errors when a user has set
// any internal resource field in their bundle configuration.
func RejectInternalResources() bundle.Mutator {
	return &rejectInternalResources{}
}

func (m *rejectInternalResources) Name() string {
	return "RejectInternalResources"
}

func (m *rejectInternalResources) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	var diags diag.Diagnostics

	var internalResourceKeys []string
	// collect all internal resource keys, only top level keys under "resources"
	structwalk.WalkType(reflect.TypeFor[config.Resources](), func(path *structpath.PatternNode, typ reflect.Type, field *reflect.StructField) bool {
		if path.Len() > 2 {
			return false
		}
		if field == nil {
			return true
		}
		tag := field.Tag.Get("bundle")
		if structtag.BundleTag(tag).Internal() {
			internalResourceKeys = append(internalResourceKeys, path.String())
		}
		return true
	})

	for _, key := range internalResourceKeys {
		v, err := dyn.GetByPath(b.Config.Value(), dyn.MustPathFromString("resources."+key))
		if err != nil {
			continue
		}
		diags = diags.Append(diag.Diagnostic{
			Severity:  diag.Error,
			Summary:   key + " is an internal resource and cannot be set in bundle configuration",
			Locations: v.Locations(),
			Paths:     []dyn.Path{dyn.MustPathFromString("resources." + key)},
		})
	}

	return diags
}

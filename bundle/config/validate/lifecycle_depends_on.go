package validate

import (
	"cmp"
	"context"
	"fmt"
	"slices"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/dyn"
)

type lifecycleDependsOn struct{}

// LifecycleDependsOn validates that every lifecycle.depends_on entry refers to an existing
// resource other than the one declaring it.
func LifecycleDependsOn() bundle.Mutator {
	return &lifecycleDependsOn{}
}

func (*lifecycleDependsOn) Name() string {
	return "validate:lifecycle_depends_on"
}

func (*lifecycleDependsOn) Apply(_ context.Context, b *bundle.Bundle) diag.Diagnostics {
	var diags diag.Diagnostics

	pattern := dyn.NewPattern(dyn.Key("resources"), dyn.AnyKey(), dyn.AnyKey(), dyn.Key("lifecycle"), dyn.Key("depends_on"))
	_, err := dyn.MapByPattern(b.Config.Value(), pattern, func(p dyn.Path, v dyn.Value) (dyn.Value, error) {
		seq, ok := v.AsSequence()
		if !ok {
			return v, nil
		}
		self := p[:3].String()
		for i, item := range seq {
			ref, ok := item.Get("resource").AsString()
			if !ok {
				continue
			}
			res, key, err := b.Config.Resources.ResolveKey(ref)
			if err == nil && "resources."+res.ResourceDescription().PluralName+"."+key == self {
				err = fmt.Errorf("resource %q cannot depend on itself", ref)
			}
			if err != nil {
				diags = diags.Append(diag.Diagnostic{
					Severity:  diag.Error,
					Summary:   "invalid lifecycle.depends_on: " + err.Error(),
					Locations: item.Locations(),
					Paths:     dyn.ToStructPaths(p.Append(dyn.Index(i), dyn.Key("resource"))),
				})
			}
		}
		return v, nil
	})
	if err != nil {
		diags = diags.Extend(diag.FromErr(err))
	}
	// MapByPattern visits resources in random order.
	slices.SortStableFunc(diags, func(a, b diag.Diagnostic) int {
		return cmp.Compare(a.Paths[0].String(), b.Paths[0].String())
	})
	return diags
}

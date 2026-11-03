package mutator

import (
	"context"
	"slices"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/dyn/convert"
	"github.com/databricks/cli/libs/dyn/dynvar"
)

type resolveBindReferences struct{}

// ResolveBindReferences resolves ${var.*}, ${bundle.*} and ${workspace.*} references
// in the selected target's bind IDs. SelectTarget keeps the target as a typed snapshot
// and drops the targets section from the config tree, so the regular variable
// resolution never sees bind blocks.
func ResolveBindReferences() bundle.Mutator {
	return &resolveBindReferences{}
}

func (m *resolveBindReferences) Name() string {
	return "ResolveBindReferences"
}

func (m *resolveBindReferences) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	if b.Target == nil || b.Target.Bind.IsEmpty() {
		return nil
	}

	bind, err := convert.FromTyped(b.Target.Bind, dyn.NilValue)
	if err != nil {
		return diag.FromErr(err)
	}

	prefixes := make([]dyn.Path, len(defaultPrefixes))
	for i, prefix := range defaultPrefixes {
		prefixes[i] = dyn.MustPathFromString(prefix)
	}

	// Same lookup root as resolveVariableReferences: fields present in the type but
	// unset in config resolve to their empty value instead of failing.
	normalized, _ := convert.Normalize(b.Config, b.Config.Value(), convert.IncludeMissingFields)
	resolved, err := dynvar.Resolve(bind, func(path dyn.Path) (dyn.Value, error) {
		path = rewriteVarShorthand(path)
		if !slices.ContainsFunc(prefixes, path.HasPrefix) {
			return dyn.InvalidValue, dynvar.ErrSkipResolution
		}
		return lookup(normalized, path, b)
	})
	if err != nil {
		return diag.FromErr(err)
	}

	// A reference may resolve to a non-string (e.g. an integer job ID), so normalize
	// against the field types before converting back.
	resolved, diags := convert.Normalize(b.Target.Bind, resolved)
	if diags.HasError() {
		return diags
	}

	return diags.Extend(diag.FromErr(convert.ToTyped(&b.Target.Bind, resolved)))
}

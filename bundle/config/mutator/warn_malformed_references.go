package mutator

import (
	"context"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/dyn/dynvar"
)

type warnMalformedReferences struct{}

// WarnMalformedReferences returns a mutator that emits warnings for strings
// containing malformed variable references (e.g. "${foo.bar-}").
func WarnMalformedReferences() bundle.Mutator {
	return &warnMalformedReferences{}
}

func (*warnMalformedReferences) Name() string {
	return "WarnMalformedReferences"
}

func (*warnMalformedReferences) Validate(ctx context.Context, b *bundle.Bundle) error {
	return nil
}

func (*warnMalformedReferences) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	var diags diag.Diagnostics
	err := b.Config.Mutate(func(root dyn.Value) (dyn.Value, error) {
		_, err := dyn.Walk(root, func(p dyn.Path, v dyn.Value) (dyn.Value, error) {
			// Only check values with source locations to avoid false positives
			// from synthesized/computed values.
			if len(v.Locations()) == 0 {
				return v, nil
			}
			_, _, refDiags := dynvar.NewRefWithDiagnostics(v)
			diags = diags.Extend(refDiags)
			return v, nil
		})
		return root, err
	})
	if err != nil {
		diags = diags.Extend(diag.FromErr(err))
	}
	return diags
}

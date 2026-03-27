package mutator

import (
	"context"
	"errors"
	"slices"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/interpolation"
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

func (*warnMalformedReferences) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	var diags diag.Diagnostics
	err := b.Config.Mutate(func(root dyn.Value) (dyn.Value, error) {
		_, err := dyn.Walk(root, func(p dyn.Path, v dyn.Value) (dyn.Value, error) {
			// Only check values with source locations to avoid false positives
			// from synthesized/computed values.
			if len(v.Locations()) == 0 {
				return v, nil
			}

			s, ok := v.AsString()
			if !ok {
				return v, nil
			}

			_, parseErr := interpolation.Parse(s)
			if parseErr == nil {
				return v, nil
			}

			var pe *interpolation.ParseError
			if !errors.As(parseErr, &pe) {
				return v, nil
			}

			// Clone locations and adjust column with the position offset
			// so the diagnostic points to the problematic reference.
			locs := slices.Clone(v.Locations())
			if len(locs) > 0 {
				locs[0].Column += pe.Pos
			}

			diags = append(diags, diag.Diagnostic{
				Severity:  diag.Warning,
				Summary:   pe.Msg,
				Locations: locs,
				Paths:     []dyn.Path{p},
			})
			return v, nil
		})
		return root, err
	})
	if err != nil {
		diags = diags.Extend(diag.FromErr(err))
	}
	return diags
}

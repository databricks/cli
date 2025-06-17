package validate

import (
	"context"
	"fmt"
	"sort"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/internal/validation/generated"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/dyn"
)

type required struct{}

func Required() bundle.Mutator {
	return &required{}
}

func (f *required) Name() string {
	return "validate:required"
}

func (f *required) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	diags := diag.Diagnostics{}

	// Generate prefix tree for all required fields.
	trie := dyn.NewPatternTrie()
	for k := range generated.RequiredFields {
		pattern, err := dyn.NewPatternFromString(k)
		if err != nil {
			return diag.FromErr(fmt.Errorf("invalid pattern %q for required field validation: %w", k, err))
		}

		err = trie.Insert(pattern)
		if err != nil {
			return diag.FromErr(fmt.Errorf("failed to insert pattern %q into trie: %w", k, err))
		}
	}

	dyn.WalkRead(b.Config.Value(), func(p dyn.Path, v dyn.Value) error {
		// If the path is not preset in the prefix tree, we do not need to validate any required
		// fields in it.
		pattern, ok := trie.SearchPath(p)
		if !ok {
			return nil
		}

		fields := generated.RequiredFields[pattern.String()]
		for _, field := range fields {
			vv := v.Get(field)
			if vv.Kind() == dyn.KindInvalid || vv.Kind() == dyn.KindNil {
				diags = diags.Append(diag.Diagnostic{
					Severity:  diag.Warning,
					Summary:   fmt.Sprintf("required field %q is not set", field),
					Locations: v.Locations(),
					Paths:     []dyn.Path{p},
				})
			}
		}
		return nil
	})

	// for k, requiredFields := range generated.RequiredFields {
	// 	pattern, err := dyn.NewPatternFromString(k)
	// 	if err != nil {
	// 		return diag.FromErr(fmt.Errorf("invalid pattern %q for required field validation: %w", k, err))
	// 	}

	// 	// Note we only emit diagnostics for fields that are not set. If a field is set to a zero
	// 	// value we don't emit a diagnostic. This is so that we defer the interpretation of zero values
	// 	// to the server.
	// 	_, err = dyn.MapByPattern(b.Config.Value(), pattern, func(p dyn.Path, v dyn.Value) (dyn.Value, error) {
	// 		for _, requiredField := range requiredFields {
	// 			vv := v.Get(requiredField)
	// 			if vv.Kind() == dyn.KindInvalid || vv.Kind() == dyn.KindNil {
	// 				diags = diags.Append(diag.Diagnostic{
	// 					Severity:  diag.Warning,
	// 					Summary:   fmt.Sprintf("required field %q is not set", requiredField),
	// 					Locations: v.Locations(),
	// 					Paths:     []dyn.Path{p},
	// 				})
	// 			}
	// 		}
	// 		return v, nil
	// 	})
	// 	if dyn.IsExpectedMapError(err) || dyn.IsExpectedSequenceError(err) || dyn.IsExpectedMapToIndexError(err) || dyn.IsExpectedSequenceToIndexError(err) {
	// 		// No map or sequence value is set at this pattern, so we ignore it.
	// 		continue
	// 	}
	// 	if err != nil {
	// 		return diag.FromErr(err)
	// 	}
	// }

	// Sort diagnostics to make them deterministic
	sort.Slice(diags, func(i, j int) bool {
		// First sort by summary
		if diags[i].Summary != diags[j].Summary {
			return diags[i].Summary < diags[j].Summary
		}

		// Then sort by locations as a tie breaker if summaries are the same.
		iLocs := fmt.Sprintf("%v", diags[i].Locations)
		jLocs := fmt.Sprintf("%v", diags[j].Locations)
		return iLocs < jLocs
	})

	return diags
}

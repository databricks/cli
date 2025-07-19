package validate

import (
	"context"
	"fmt"
	"slices"
	"sort"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/internal/validation/generated"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/dyn"
)

type enum struct{}

func Enum() bundle.Mutator {
	return &enum{}
}

func (e *enum) Name() string {
	return "validate:enum"
}

func (e *enum) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	diags := diag.Diagnostics{}

	// Generate prefix tree for all enum fields.
	trie := &dyn.TrieNode{}
	for k := range generated.EnumFields {
		pattern, err := dyn.NewPatternFromString(k)
		if err != nil {
			return diag.FromErr(fmt.Errorf("invalid pattern %q for enum field validation: %w", k, err))
		}

		err = trie.Insert(pattern)
		if err != nil {
			return diag.FromErr(fmt.Errorf("failed to insert pattern %q into trie: %w", k, err))
		}
	}

	err := dyn.WalkReadOnly(b.Config.Value(), func(p dyn.Path, v dyn.Value) error {
		// If the path is not found in the prefix tree, we do not need to validate any enum
		// fields in it.
		pattern, ok := trie.SearchPath(p)
		if !ok {
			return nil
		}

		// Get the allowed enum values for this pattern
		allowedValues := generated.EnumFields[pattern.String()]

		// Check if the current value is valid for this enum field
		if v.Kind() == dyn.KindString {
			actualValue := v.MustString()
			if !slices.Contains(allowedValues, actualValue) {
				cloneP := slices.Clone(p)
				diags = diags.Append(diag.Diagnostic{
					Severity:  diag.Warning,
					Summary:   fmt.Sprintf("invalid value %q for enum field. Expected one of: %v", actualValue, allowedValues),
					Locations: v.Locations(),
					Paths:     []dyn.Path{cloneP},
				})
			}
		}
		return nil
	})
	if err != nil {
		return diag.FromErr(err)
	}

	// Sort diagnostics to make them deterministic
	sort.Slice(diags, func(i, j int) bool {
		// First sort by summary
		if diags[i].Summary != diags[j].Summary {
			return diags[i].Summary < diags[j].Summary
		}

		// Finally sort by locations as a tie breaker if summaries are the same.
		iLocs := fmt.Sprintf("%v", diags[i].Locations)
		jLocs := fmt.Sprintf("%v", diags[j].Locations)
		return iLocs < jLocs
	})

	return diags
}

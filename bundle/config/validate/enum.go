package validate

import (
	"cmp"
	"context"
	"fmt"
	"slices"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/internal/validation/generated"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/dyn/dynvar"
)

type enum struct{}

func Enum() bundle.Mutator {
	return &enum{}
}

func (f *enum) Name() string {
	return "validate:enum"
}

func (f *enum) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
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

		// Get the string value for comparison
		strValue, ok := v.AsString()
		if !ok {
			return nil
		}

		// Skip validation for values containing variable references (e.g.
		// ${resources.jobs.my_job.id}) since they are not yet resolved.
		if dynvar.ContainsVariableReference(strValue) {
			return nil
		}

		// Get valid values for this pattern
		validValues := generated.EnumFields[pattern.String()]

		// Check if the value is in the list of valid enum values
		validValue := slices.Contains(validValues, strValue)

		if !validValue {
			// p is a slice of path components. We need to clone it before using it in diagnostics
			// since the WalkReadOnly function will mutate it while walking the config tree.
			cloneP := slices.Clone(p)

			diags = diags.Append(diag.Diagnostic{
				Severity:  diag.Warning,
				Summary:   fmt.Sprintf("invalid value %q for enum field. Valid values are %v", strValue, validValues),
				Locations: v.Locations(),
				Paths:     []dyn.Path{cloneP},
			})
		}

		return nil
	})
	if err != nil {
		return diag.FromErr(err)
	}

	// Sort diagnostics to make them deterministic
	slices.SortFunc(diags, func(a, b diag.Diagnostic) int {
		// First sort by summary
		if n := cmp.Compare(a.Summary, b.Summary); n != 0 {
			return n
		}

		// Then sort by locations as a tie breaker if summaries are the same.
		return cmp.Compare(fmt.Sprintf("%v", a.Locations), fmt.Sprintf("%v", b.Locations))
	})

	return diags
}

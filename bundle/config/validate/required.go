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
	"github.com/databricks/cli/libs/logdiag"
)

type required struct{}

func Required() bundle.Mutator {
	return &required{}
}

func (f *required) Name() string {
	return "validate:required"
}

// Warn for missing fields, based on annotations in the Go SDK / OpenAPI spec.
func warnForMissingFields(ctx context.Context, b *bundle.Bundle) error {
	diags := diag.Diagnostics{}

	// Generate prefix tree for all required fields.
	trie := &dyn.TrieNode{}
	for k := range generated.RequiredFields {
		pattern, err := dyn.NewPatternFromString(k)
		if err != nil {
			return fmt.Errorf("invalid pattern %q for required field validation: %w", k, err)
		}

		err = trie.Insert(pattern)
		if err != nil {
			return fmt.Errorf("failed to insert pattern %q into trie: %w", k, err)
		}
	}

	err := dyn.WalkReadOnly(b.Config.Value(), func(p dyn.Path, v dyn.Value) error {
		// If the path is not found in the prefix tree, we do not need to validate any required
		// fields in it.
		pattern, ok := trie.SearchPath(p)
		if !ok {
			return nil
		}

		cloneP := slices.Clone(p)

		fields := generated.RequiredFields[pattern.String()]
		for _, field := range fields {
			vv := v.Get(field)
			if vv.Kind() == dyn.KindInvalid || vv.Kind() == dyn.KindNil {
				diags = diags.Append(diag.Diagnostic{
					Severity:  diag.Warning,
					Summary:   fmt.Sprintf("required field %q is not set", field),
					Locations: v.Locations(),
					Paths:     []dyn.Path{cloneP},
				})
			}
		}
		return nil
	})
	if err != nil {
		return err
	}

	// Sort diagnostics to make them deterministic
	slices.SortFunc(diags, func(a, b diag.Diagnostic) int {
		// First sort by summary
		if n := cmp.Compare(a.Summary, b.Summary); n != 0 {
			return n
		}

		// Finally sort by locations as a tie breaker if summaries are the same.
		return cmp.Compare(fmt.Sprintf("%v", a.Locations), fmt.Sprintf("%v", b.Locations))
	})

	for _, d := range diags {
		logdiag.LogDiag(ctx, d)
	}

	return nil
}

// Bespoke code to error for fields that are not marked as required in the Go SDK / OpenAPI spec.
func errorForMissingFields(ctx context.Context, b *bundle.Bundle) error {
	// Dashboards should always have a name and warehouse_id.
	var nameLocations []dyn.Location
	var namePaths []dyn.Path
	var warehouseIdLocations []dyn.Location
	var warehouseIdPaths []dyn.Path

	for key, dashboard := range b.Config.Resources.Dashboards {
		if dashboard.DisplayName == "" {
			nameLocations = append(nameLocations, b.Config.GetLocations("resources.dashboards."+key)...)
			namePaths = append(namePaths, dyn.MustPathFromString("resources.dashboards."+key))
		}
		if dashboard.WarehouseId == "" {
			warehouseIdLocations = append(warehouseIdLocations, b.Config.GetLocations("resources.dashboards."+key)...)
			warehouseIdPaths = append(warehouseIdPaths, dyn.MustPathFromString("resources.dashboards."+key))
		}
	}

	var diags diag.Diagnostics
	if len(nameLocations) > 0 {
		diags = diags.Append(diag.Diagnostic{
			Severity:  diag.Error,
			Summary:   "dashboard display_name is required",
			Locations: nameLocations,
			Paths:     namePaths,
		})
	}
	if len(warehouseIdLocations) > 0 {
		diags = diags.Append(diag.Diagnostic{
			Severity:  diag.Error,
			Summary:   "dashboard warehouse_id is required",
			Locations: warehouseIdLocations,
			Paths:     warehouseIdPaths,
		})
	}

	return logdiag.Flush(ctx, diags)
}

func (f *required) Apply(ctx context.Context, b *bundle.Bundle) error {
	if err := errorForMissingFields(ctx, b); err != nil {
		return err
	}
	return warnForMissingFields(ctx, b)
}

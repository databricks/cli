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

type required struct{}

func Required() bundle.Mutator {
	return &required{}
}

func (f *required) Name() string {
	return "validate:required"
}

// Warn for missing fields, based on annotations in the Go SDK / OpenAPI spec.
func warnForMissingFields(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	diags := diag.Diagnostics{}

	// Generate prefix tree for all required fields.
	trie := &dyn.TrieNode{}
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

// Bespoke code to error for fields that are not marked as required in the Go SDK / OpenAPI spec.
func errorForMissingFields(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	// Dashboards should always have a name and warehouse_id.
	nameLocations := []dyn.Location{}
	namePaths := []dyn.Path{}
	warehouseIdLocations := []dyn.Location{}
	warehouseIdPaths := []dyn.Path{}
	diags := diag.Diagnostics{}
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

	if len(nameLocations) > 0 {
		diags = diags.Append(diag.Diagnostic{
			Severity:  diag.Error,
			Summary:   "name is required for dashboards",
			Locations: nameLocations,
			Paths:     namePaths,
		})
	}
	if len(warehouseIdLocations) > 0 {
		diags = diags.Append(diag.Diagnostic{
			Severity:  diag.Error,
			Summary:   "warehouse_id is required for dashboards",
			Locations: warehouseIdLocations,
			Paths:     warehouseIdPaths,
		})
	}

	return diags
}

func (f *required) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	diags := errorForMissingFields(ctx, b)
	if diags.HasError() {
		return diags
	}
	diags = diags.Extend(warnForMissingFields(ctx, b))
	return diags
}

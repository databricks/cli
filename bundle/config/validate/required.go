package validate

import (
	"cmp"
	"context"
	"fmt"
	"slices"
	"strings"

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

	sortDiagnostics(diags)

	return diags
}

// sortDiagnostics orders diagnostics deterministically, since they are collected
// by walking maps with random iteration order.
func sortDiagnostics(diags diag.Diagnostics) {
	slices.SortFunc(diags, func(a, b diag.Diagnostic) int {
		// First sort by summary
		if n := cmp.Compare(a.Summary, b.Summary); n != 0 {
			return n
		}

		// Finally sort by locations as a tie breaker if summaries are the same.
		return cmp.Compare(fmt.Sprintf("%v", a.Locations), fmt.Sprintf("%v", b.Locations))
	})
}

// Bespoke code to error for fields that are not marked as required in the Go SDK / OpenAPI spec.
func errorForMissingFields(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	// Dashboards should always have a name and warehouse_id.
	var nameLocations []dyn.Location
	var namePaths []dyn.Path
	var warehouseIdLocations []dyn.Location
	var warehouseIdPaths []dyn.Path

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

	// sql_warehouses.name is optional in the SDK (json:"name,omitempty") but required
	// by the backend, which rejects whitespace-only names (name.trim.nonEmpty).
	for key, warehouse := range b.Config.Resources.SqlWarehouses {
		if strings.TrimSpace(warehouse.Name) == "" {
			path := "resources.sql_warehouses." + key
			diags = diags.Append(diag.Diagnostic{
				Severity:  diag.Error,
				Summary:   "sql_warehouse name is required",
				Locations: b.Config.GetLocations(path),
				Paths:     []dyn.Path{dyn.MustPathFromString(path)},
			})
		}
	}

	sortDiagnostics(diags)

	return diags
}

// errorForInvalidGrants errors for grants the backend rejects or that never converge:
// a missing principal is rejected, and an empty privileges list re-plans forever because
// the backend drops principals with no privileges. Erroring here (rather than warning)
// avoids a partial deploy where the securable is created before the grants call fails.
// Grants exist on every securable, so match any resource type.
func errorForInvalidGrants(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	diags := diag.Diagnostics{}

	_, err := dyn.MapByPattern(
		b.Config.Value(),
		dyn.NewPattern(dyn.Key("resources"), dyn.AnyKey(), dyn.AnyKey(), dyn.Key("grants"), dyn.AnyIndex()),
		func(p dyn.Path, v dyn.Value) (dyn.Value, error) {
			if isMissingOrEmptyString(v.Get("principal")) {
				diags = diags.Append(diag.Diagnostic{
					Severity:  diag.Error,
					Summary:   "grant principal is required",
					Locations: v.Locations(),
					Paths:     []dyn.Path{slices.Clone(p)},
				})
			}
			if isMissingOrEmptySequence(v.Get("privileges")) {
				diags = diags.Append(diag.Diagnostic{
					Severity:  diag.Error,
					Summary:   "grant privileges is required",
					Locations: v.Locations(),
					Paths:     []dyn.Path{slices.Clone(p)},
				})
			}
			return v, nil
		},
	)
	if err != nil {
		return diag.FromErr(err)
	}

	sortDiagnostics(diags)

	return diags
}

// isMissingOrEmptyString reports whether v is unset, null, or an empty string.
func isMissingOrEmptyString(v dyn.Value) bool {
	switch v.Kind() {
	case dyn.KindInvalid, dyn.KindNil:
		return true
	case dyn.KindString:
		return v.MustString() == ""
	default:
		return false
	}
}

// isMissingOrEmptySequence reports whether v is unset, null, or an empty sequence.
func isMissingOrEmptySequence(v dyn.Value) bool {
	switch v.Kind() {
	case dyn.KindInvalid, dyn.KindNil:
		return true
	case dyn.KindSequence:
		return len(v.MustSequence()) == 0
	default:
		return false
	}
}

func (f *required) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	diags := errorForMissingFields(ctx, b)
	diags = diags.Extend(errorForInvalidGrants(ctx, b))
	if diags.HasError() {
		return diags
	}
	diags = diags.Extend(warnForMissingFields(ctx, b))
	return diags
}

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

	// sql_warehouses.name is required by the backend but optional in the SDK
	// (json:"name,omitempty"), so warnForMissingFields never flags it.
	_, err := dyn.MapByPattern(
		b.Config.Value(),
		dyn.NewPattern(dyn.Key("resources"), dyn.Key("sql_warehouses"), dyn.AnyKey()),
		func(p dyn.Path, v dyn.Value) (dyn.Value, error) {
			if isMissingOrBlankString(v.Get("name")) {
				diags = diags.Append(diag.Diagnostic{
					Severity:  diag.Error,
					Summary:   "sql_warehouse name is required",
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

// Warn for fields the backend appears to require but that are optional in the SDK,
// where the contract is unconfirmed. Warnings, not errors, so a wrong guess can't
// block a valid deploy.
func warnForMissingBackendFields(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	diags := diag.Diagnostics{}

	// grants[*].principal is optional in the SDK (json:"principal,omitempty").
	// Grants exist on every securable, so match any.
	_, err := dyn.MapByPattern(
		b.Config.Value(),
		dyn.NewPattern(dyn.Key("resources"), dyn.AnyKey(), dyn.AnyKey(), dyn.Key("grants"), dyn.AnyIndex()),
		func(p dyn.Path, v dyn.Value) (dyn.Value, error) {
			if isMissingOrEmptyString(v.Get("principal")) {
				diags = diags.Append(diag.Diagnostic{
					Severity:  diag.Warning,
					Summary:   "grant principal is required",
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

// isMissingOrBlankString reports whether v is unset, null, or blank (whitespace
// only), mirroring the sql_warehouses backend check name.trim.nonEmpty.
func isMissingOrBlankString(v dyn.Value) bool {
	if isMissingOrEmptyString(v) {
		return true
	}
	return v.Kind() == dyn.KindString && strings.TrimSpace(v.MustString()) == ""
}

func (f *required) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	diags := errorForMissingFields(ctx, b)
	if diags.HasError() {
		return diags
	}
	diags = diags.Extend(warnForMissingFields(ctx, b))
	diags = diags.Extend(warnForMissingBackendFields(ctx, b))
	return diags
}

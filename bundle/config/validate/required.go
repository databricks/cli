package validate

import (
	"cmp"
	"context"
	"fmt"
	"slices"

	"github.com/databricks/cli/bundle"
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

// sortDiagnostics orders diagnostics deterministically, since they are collected
// by walking maps with random iteration order.
func sortDiagnostics(diags diag.Diagnostics) {
	slices.SortFunc(diags, func(a, b diag.Diagnostic) int {
		// Keep errors ahead of warnings, then sort each group by summary.
		if n := cmp.Compare(a.Severity, b.Severity); n != 0 {
			return n
		}
		if n := cmp.Compare(a.Summary, b.Summary); n != 0 {
			return n
		}

		// Then sort by locations as a tie breaker if summaries are the same.
		if n := cmp.Compare(fmt.Sprintf("%v", a.Locations), fmt.Sprintf("%v", b.Locations)); n != 0 {
			return n
		}

		// Sibling entries can share a location; fall back to path for a stable order.
		return cmp.Compare(fmt.Sprintf("%v", a.Paths), fmt.Sprintf("%v", b.Paths))
	})
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

// errorForInvalidSecretScopePermissions errors when a permission names no principal.
// Wrong-typed values are already empty here (normalization only warns and drops them).
func errorForInvalidSecretScopePermissions(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	diags := diag.Diagnostics{}

	for key, scope := range b.Config.Resources.SecretScopes {
		for i, perm := range scope.Permissions {
			if perm.UserName != "" || perm.GroupName != "" || perm.ServicePrincipalName != "" {
				continue
			}
			path := fmt.Sprintf("resources.secret_scopes.%s.permissions[%d]", key, i)
			// ApplyBundlePermissions rebuilds permissions via convert.FromTyped and drops
			// per-entry locations, so point at the scope.
			diags = diags.Append(diag.Diagnostic{
				Severity:  diag.Error,
				Summary:   "secret scope permission principal is required",
				Detail:    "Set one of user_name, group_name or service_principal_name",
				Locations: b.Config.GetLocations("resources.secret_scopes." + key),
				Paths:     []dyn.Path{dyn.MustPathFromString(path)},
			})
		}
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
	diags := validateRequiredFields(ctx, b)
	diags = diags.Extend(errorForInvalidGrants(ctx, b))
	diags = diags.Extend(errorForInvalidSecretScopePermissions(ctx, b))
	diags = diags.Extend(errorForIncompletePipelineLibraries(ctx, b))
	sortDiagnostics(diags)
	return diags
}

package validate

import (
	"context"
	"fmt"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/auth"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/dyn/dynvar"
)

type noInterpolationInAuthConfig struct{}

func NoInterpolationInAuthConfig() bundle.Mutator {
	return &noInterpolationInAuthConfig{}
}

func (f *noInterpolationInAuthConfig) Name() string {
	return "validate:interpolation_in_auth_config"
}

func (f *noInterpolationInAuthConfig) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	authFields := []string{
		// Generic attributes.
		"host",
		"profile",
		"auth_type",
		"metadata_service_url",

		// OAuth specific attributes.
		"client_id",

		// Google specific attributes.
		"google_service_account",

		// Azure specific attributes.
		"azure_resource_id",
		"azure_use_msi",
		"azure_client_id",
		"azure_tenant_id",
		"azure_environment",
		"azure_login_app_id",
	}

	diags := diag.Diagnostics{}

	for _, fieldName := range authFields {
		p := dyn.NewPath(dyn.Key("workspace"), dyn.Key(fieldName))
		v, err := dyn.GetByPath(b.Config.Value(), p)
		if dyn.IsNoSuchKeyError(err) {
			continue
		}
		if err != nil {
			return diag.FromErr(err)
		}

		vv, ok := v.AsString()
		if !ok {
			continue
		}

		// Check if the field contains interpolation.
		if dynvar.ContainsVariableReference(vv) {
			envVar, ok := auth.GetEnvFor(fieldName)
			if !ok {
				continue
			}

			diags = append(diags, diag.Diagnostic{
				Severity: diag.Warning,
				Summary:  "Variable interpolation is not supported for fields that configure authentication",
				Detail: fmt.Sprintf(`Interpolation is not supported for the field %s. Please set
the %s environment variable if you wish to configure this field at runtime.`, p.String(), envVar),
				Locations: v.Locations(),
				Paths:     []dyn.Path{p},
			})
		}
	}

	return diags
}

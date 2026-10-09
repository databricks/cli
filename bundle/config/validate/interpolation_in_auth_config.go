package validate

import (
	"context"
	"fmt"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/auth"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/structs/structpath"
	"github.com/databricks/cli/libs/structs/structvar"
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

		// Unified host specific attributes.
		"account_id",
		"workspace_id",
	}

	diags := diag.Diagnostics{}

	for _, fieldName := range authFields {
		p := structpath.NewPath(nil, "workspace", fieldName)
		v := b.Config.View().Lookup(p)
		if !v.IsValid() {
			continue
		}

		vv, ok := v.AsString()
		if !ok {
			continue
		}

		// Check if the field contains interpolation.
		if structvar.ContainsVariableReference(vv) {
			envVar, ok := auth.GetEnvFor(fieldName)
			if !ok {
				continue
			}

			diags = append(diags, diag.Diagnostic{
				Severity: diag.Warning,
				Summary:  "Variable interpolation is not supported for fields that configure authentication",
				Detail: fmt.Sprintf(`Interpolation is not supported for the field %s. Please set
the %s environment variable if you wish to configure this field at runtime.`, p.String(), envVar),
				Paths: []*structpath.PathNode{p},
			})
		}
	}

	return diags
}

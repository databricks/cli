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
	w := &b.Config.Workspace
	authFields := []struct {
		name  string
		value string
	}{
		// Generic attributes.
		{"host", w.Host},
		{"profile", w.Profile},
		{"auth_type", w.AuthType},
		{"metadata_service_url", w.MetadataServiceURL},

		// OAuth specific attributes.
		{"client_id", w.ClientID},

		// Google specific attributes.
		{"google_service_account", w.GoogleServiceAccount},

		// Azure specific attributes. "azure_resource_id" used to be listed here, but it
		// is not a workspace key (the field is azure_workspace_resource_id), so it was
		// never checked.
		{"azure_use_msi", b.Config.ReferenceAt(structpath.NewPath(nil, "workspace", "azure_use_msi"))},
		{"azure_client_id", w.AzureClientID},
		{"azure_tenant_id", w.AzureTenantID},
		{"azure_environment", w.AzureEnvironment},
		{"azure_login_app_id", w.AzureLoginAppID},

		// Unified host specific attributes.
		{"account_id", w.AccountID},
		{"workspace_id", w.WorkspaceID},
	}

	diags := diag.Diagnostics{}

	for _, field := range authFields {
		// Check if the field contains interpolation.
		if !structvar.ContainsVariableReference(field.value) {
			continue
		}
		envVar, ok := auth.GetEnvFor(field.name)
		if !ok {
			continue
		}

		p := structpath.NewPath(nil, "workspace", field.name)
		diags = append(diags, diag.Diagnostic{
			Severity: diag.Warning,
			Summary:  "Variable interpolation is not supported for fields that configure authentication",
			Detail: fmt.Sprintf(`Interpolation is not supported for the field %s. Please set
the %s environment variable if you wish to configure this field at runtime.`, p.String(), envVar),
			Locations: b.Config.GetLocationsOfExact(p),
			Paths:     []*structpath.PathNode{p},
		})
	}

	return diags
}

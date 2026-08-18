package validate

import (
	"context"
	"fmt"
	"strings"
	"unicode"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/dyn"
)

// errorForInvalidIdentifiers rejects empty, blank, or control-character identifiers.
func errorForInvalidIdentifiers(_ context.Context, b *bundle.Bundle) diag.Diagnostics {
	diags := diag.Diagnostics{}

	for key, model := range b.Config.Resources.Models {
		diags = diags.Extend(identifierDiag(b, "model", "name", model.Name, "resources.models."+key, true))
	}
	for key, catalog := range b.Config.Resources.Catalogs {
		diags = diags.Extend(identifierDiag(b, "catalog", "name", catalog.Name, "resources.catalogs."+key, true))
	}
	for key, schema := range b.Config.Resources.Schemas {
		path := "resources.schemas." + key
		diags = diags.Extend(identifierDiag(b, "schema", "name", schema.Name, path, true))
		diags = diags.Extend(identifierDiag(b, "schema", "catalog_name", schema.CatalogName, path, false))
	}
	for key, volume := range b.Config.Resources.Volumes {
		path := "resources.volumes." + key
		diags = diags.Extend(identifierDiag(b, "volume", "name", volume.Name, path, true))
		diags = diags.Extend(identifierDiag(b, "volume", "catalog_name", volume.CatalogName, path, false))
		diags = diags.Extend(identifierDiag(b, "volume", "schema_name", volume.SchemaName, path, false))
	}
	for key, loc := range b.Config.Resources.ExternalLocations {
		diags = diags.Extend(identifierDiag(b, "external_location", "name", loc.Name, "resources.external_locations."+key, true))
	}
	for key, model := range b.Config.Resources.RegisteredModels {
		path := "resources.registered_models." + key
		diags = diags.Extend(identifierDiag(b, "registered_model", "name", model.Name, path, true))
		diags = diags.Extend(identifierDiag(b, "registered_model", "catalog_name", model.CatalogName, path, false))
		diags = diags.Extend(identifierDiag(b, "registered_model", "schema_name", model.SchemaName, path, false))
	}
	for key, endpoint := range b.Config.Resources.ModelServingEndpoints {
		diags = diags.Extend(identifierDiag(b, "model_serving_endpoint", "name", endpoint.Name, "resources.model_serving_endpoints."+key, true))
	}

	sortDiagnostics(diags)
	return diags
}

// errorForIncompletePipelineLibraries rejects file, notebook, and glob entries without paths.
func errorForIncompletePipelineLibraries(_ context.Context, b *bundle.Bundle) diag.Diagnostics {
	diags := diag.Diagnostics{}

	for key, pipeline := range b.Config.Resources.Pipelines {
		for i, lib := range pipeline.Libraries {
			base := fmt.Sprintf("resources.pipelines.%s.libraries[%d]", key, i)
			if lib.File != nil && strings.TrimSpace(lib.File.Path) == "" {
				diags = diags.Append(diag.Diagnostic{
					Severity:  diag.Error,
					Summary:   "pipeline library file path is required",
					Locations: b.Config.GetLocations(base),
					Paths:     []dyn.Path{dyn.MustPathFromString(base + ".file")},
				})
			}
			if lib.Notebook != nil && strings.TrimSpace(lib.Notebook.Path) == "" {
				diags = diags.Append(diag.Diagnostic{
					Severity:  diag.Error,
					Summary:   "pipeline library notebook path is required",
					Locations: b.Config.GetLocations(base),
					Paths:     []dyn.Path{dyn.MustPathFromString(base + ".notebook")},
				})
			}
			if lib.Glob != nil && strings.TrimSpace(lib.Glob.Include) == "" {
				diags = diags.Append(diag.Diagnostic{
					Severity:  diag.Error,
					Summary:   "pipeline library glob include is required",
					Locations: b.Config.GetLocations(base),
					Paths:     []dyn.Path{dyn.MustPathFromString(base + ".glob")},
				})
			}
		}
	}

	sortDiagnostics(diags)
	return diags
}

func identifierDiag(b *bundle.Bundle, resource, field, value, resourcePath string, required bool) diag.Diagnostics {
	reason := invalidIdentifierReason(value, required)
	if reason == "" {
		return nil
	}

	fieldPath := resourcePath + "." + field
	locations := b.Config.GetLocations(fieldPath)
	if len(locations) == 0 {
		locations = b.Config.GetLocations(resourcePath)
	}

	return diag.Diagnostics{{
		Severity:  diag.Error,
		Summary:   fmt.Sprintf("%s %s %s", resource, field, reason),
		Locations: locations,
		Paths:     []dyn.Path{dyn.MustPathFromString(fieldPath)},
	}}
}

func invalidIdentifierReason(value string, required bool) string {
	if value == "" {
		if !required {
			return ""
		}
		return "is required"
	}
	if containsControlCharacter(value) {
		return "must not contain control characters"
	}
	if strings.TrimSpace(value) == "" {
		return "must not be blank"
	}
	return ""
}

func containsControlCharacter(s string) bool {
	return strings.ContainsFunc(s, unicode.IsControl)
}

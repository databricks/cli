package validate

import (
	"context"
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/dyn"
)

// validateIdentifiers rejects values that the backend cannot use as identifiers.
func validateIdentifiers(_ context.Context, b *bundle.Bundle) diag.Diagnostics {
	var diags diag.Diagnostics

	bundlePath := dyn.NewPath(dyn.Key("bundle"))
	if pathExists(b, bundlePath) {
		diags = diags.Extend(validateIdentifier(b, bundlePath, "name", b.Config.Bundle.Name, true))
	}

	for key, resource := range b.Config.Resources.Alerts {
		diags = diags.Extend(validateResourceIdentifier(b, "alerts", key, "display_name", resource.DisplayName, true))
	}
	for key, resource := range b.Config.Resources.Apps {
		diags = diags.Extend(validateResourceIdentifier(b, "apps", key, "name", resource.Name, true))
	}
	for key, resource := range b.Config.Resources.Catalogs {
		diags = diags.Extend(validateResourceIdentifier(b, "catalogs", key, "name", resource.Name, true))
	}
	for key, resource := range b.Config.Resources.Dashboards {
		diags = diags.Extend(validateResourceIdentifier(b, "dashboards", key, "display_name", resource.DisplayName, true))
	}
	for key, resource := range b.Config.Resources.DatabaseCatalogs {
		diags = diags.Extend(validateResourceIdentifier(b, "database_catalogs", key, "database_instance_name", resource.DatabaseInstanceName, false))
		diags = diags.Extend(validateResourceIdentifier(b, "database_catalogs", key, "database_name", resource.DatabaseName, false))
		diags = diags.Extend(validateResourceIdentifier(b, "database_catalogs", key, "name", resource.Name, true))
	}
	for key, resource := range b.Config.Resources.DatabaseInstances {
		diags = diags.Extend(validateResourceIdentifier(b, "database_instances", key, "name", resource.Name, true))
	}
	for key, resource := range b.Config.Resources.Experiments {
		diags = diags.Extend(validateResourceIdentifier(b, "experiments", key, "name", resource.Name, true))
	}
	for key, resource := range b.Config.Resources.ExternalLocations {
		diags = diags.Extend(validateResourceIdentifier(b, "external_locations", key, "credential_name", resource.CredentialName, false))
		diags = diags.Extend(validateResourceIdentifier(b, "external_locations", key, "name", resource.Name, true))
	}
	for key, resource := range b.Config.Resources.InstancePools {
		diags = diags.Extend(validateResourceIdentifier(b, "instance_pools", key, "instance_pool_name", resource.InstancePoolName, true))
	}
	for key, resource := range b.Config.Resources.ModelServingEndpoints {
		diags = diags.Extend(validateResourceIdentifier(b, "model_serving_endpoints", key, "name", resource.Name, true))
	}
	for key, resource := range b.Config.Resources.Models {
		diags = diags.Extend(validateResourceIdentifier(b, "models", key, "name", resource.Name, true))
	}
	for key, resource := range b.Config.Resources.QualityMonitors {
		diags = diags.Extend(validateResourceIdentifier(b, "quality_monitors", key, "output_schema_name", resource.OutputSchemaName, false))
		diags = diags.Extend(validateResourceIdentifier(b, "quality_monitors", key, "table_name", resource.TableName, false))
	}
	for key, resource := range b.Config.Resources.RegisteredModels {
		diags = diags.Extend(validateResourceIdentifier(b, "registered_models", key, "catalog_name", resource.CatalogName, false))
		diags = diags.Extend(validateResourceIdentifier(b, "registered_models", key, "name", resource.Name, true))
		diags = diags.Extend(validateResourceIdentifier(b, "registered_models", key, "schema_name", resource.SchemaName, false))
	}
	for key, resource := range b.Config.Resources.Schemas {
		diags = diags.Extend(validateResourceIdentifier(b, "schemas", key, "catalog_name", resource.CatalogName, false))
		diags = diags.Extend(validateResourceIdentifier(b, "schemas", key, "name", resource.Name, true))
	}
	for key, resource := range b.Config.Resources.Secrets {
		diags = diags.Extend(validateResourceIdentifier(b, "secrets", key, "catalog_name", resource.CatalogName, false))
		diags = diags.Extend(validateResourceIdentifier(b, "secrets", key, "name", resource.Name, true))
		diags = diags.Extend(validateResourceIdentifier(b, "secrets", key, "schema_name", resource.SchemaName, false))
	}
	for key, resource := range b.Config.Resources.SecretScopes {
		diags = diags.Extend(validateResourceIdentifier(b, "secret_scopes", key, "name", resource.Name, true))
	}
	for key, resource := range b.Config.Resources.SqlWarehouses {
		diags = diags.Extend(validateResourceIdentifier(b, "sql_warehouses", key, "name", resource.Name, true))
	}
	for key, resource := range b.Config.Resources.SyncedDatabaseTables {
		diags = diags.Extend(validateResourceIdentifier(b, "synced_database_tables", key, "name", resource.Name, true))
	}
	for key, resource := range b.Config.Resources.VectorSearchEndpoints {
		diags = diags.Extend(validateResourceIdentifier(b, "vector_search_endpoints", key, "name", resource.Name, true))
	}
	for key, resource := range b.Config.Resources.VectorSearchIndexes {
		diags = diags.Extend(validateResourceIdentifier(b, "vector_search_indexes", key, "endpoint_name", resource.EndpointName, false))
		diags = diags.Extend(validateResourceIdentifier(b, "vector_search_indexes", key, "name", resource.Name, true))
	}
	for key, resource := range b.Config.Resources.Volumes {
		diags = diags.Extend(validateResourceIdentifier(b, "volumes", key, "catalog_name", resource.CatalogName, false))
		diags = diags.Extend(validateResourceIdentifier(b, "volumes", key, "name", resource.Name, true))
		diags = diags.Extend(validateResourceIdentifier(b, "volumes", key, "schema_name", resource.SchemaName, false))
	}

	return diags
}

func missingIdentifierIsError(pattern, field string) bool {
	switch pattern {
	case "bundle":
		return field == "name"
	case "resources.alerts.*", "resources.dashboards.*":
		return field == "display_name"
	case "resources.instance_pools.*":
		return field == "instance_pool_name"
	case "resources.apps.*",
		"resources.catalogs.*",
		"resources.database_catalogs.*",
		"resources.database_instances.*",
		"resources.experiments.*",
		"resources.external_locations.*",
		"resources.model_serving_endpoints.*",
		"resources.models.*",
		"resources.registered_models.*",
		"resources.schemas.*",
		"resources.secret_scopes.*",
		"resources.secrets.*",
		"resources.sql_warehouses.*",
		"resources.synced_database_tables.*",
		"resources.vector_search_endpoints.*",
		"resources.vector_search_indexes.*",
		"resources.volumes.*":
		return field == "name"
	default:
		return false
	}
}

func validateResourceIdentifier(b *bundle.Bundle, resourceType, key, field, value string, required bool) diag.Diagnostics {
	resourcePath := dyn.NewPath(
		dyn.Key("resources"),
		dyn.Key(resourceType),
		dyn.Key(key),
	)
	return validateIdentifier(b, resourcePath, field, value, required)
}

func validateIdentifier(b *bundle.Bundle, resourcePath dyn.Path, field, value string, required bool) diag.Diagnostics {
	fieldPath := resourcePath.Append(dyn.Key(field))
	locations := locationsAtPath(b, fieldPath)
	if value == "" && !required && !pathExists(b, fieldPath) {
		return nil
	}

	reason, detail := invalidIdentifierReason(value)
	if reason == "" {
		return nil
	}
	if len(locations) == 0 {
		locations = locationsAtPath(b, resourcePath)
	}
	return diag.Diagnostics{{
		Severity:  diag.Error,
		Summary:   fmt.Sprintf("%s %s %s", requiredObjectName(resourcePath), field, reason),
		Detail:    detail,
		Locations: locations,
		Paths:     []dyn.Path{fieldPath},
	}}
}

// locationsAtPath avoids GetLocations: string paths do not round-trip keys with '[' or '.'.
func locationsAtPath(b *bundle.Bundle, path dyn.Path) []dyn.Location {
	value, err := dyn.GetByPath(b.Config.Value(), path)
	if err != nil {
		return nil
	}
	return value.Locations()
}

func pathExists(b *bundle.Bundle, path dyn.Path) bool {
	_, err := dyn.GetByPath(b.Config.Value(), path)
	return err == nil
}

func invalidIdentifierReason(value string) (reason, detail string) {
	if value == "" {
		return "is required", ""
	}
	if i := strings.IndexFunc(value, unicode.IsControl); i >= 0 {
		r, _ := utf8.DecodeRuneInString(value[i:])
		return "must not contain control characters", fmt.Sprintf("The value contains %U at byte offset %d", r, i)
	}
	if strings.TrimSpace(value) == "" {
		return "must not be blank", ""
	}
	return "", ""
}

func requiredObjectName(path dyn.Path) string {
	if len(path) == 1 {
		return path[0].Key()
	}
	plural := path[1].Key()
	if desc, ok := config.SupportedResources()[plural]; ok && desc.SingularName != "" {
		return desc.SingularName
	}
	return plural
}

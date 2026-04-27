// Package validate contains ucm's validation mutators.
//
// These run in two waves:
//
//   - Raw-config (pre-interpolation): required-field checks, UC naming rules,
//     duplicate resource-key detection. Composed by [All] and wired into
//     phases.Validate / phases.PolicyCheck.
//   - Post-interpolation: reference-closure checks that run after
//     ResolveVariableReferencesOnlyResources (and later, variable resolution)
//     so the validator sees concrete values.
package validate

import (
	"context"
	"fmt"
	"sort"

	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/ucm"
)

// RequiredFields errors on UC resources missing their mandatory fields.
//
// Required-field matrix (kept in-code rather than sourced from the SDK
// annotations so diagnostics are terse and domain-specific):
//
//   - catalog:             name
//   - schema:              name, catalog
//   - grant:               principal, privileges (non-empty), securable.type, securable.name
//   - storage_credential:  name + exactly one cloud-identity sub-struct
//   - external_location:   name, url, credential_name
//   - volume:              name, catalog_name, schema_name, volume_type
//   - connection:          name, connection_type, options (non-empty)
func RequiredFields() ucm.Mutator { return &requiredFields{} }

type requiredFields struct{}

func (m *requiredFields) Name() string { return "validate:required_fields" }

func (m *requiredFields) Apply(_ context.Context, u *ucm.Ucm) diag.Diagnostics {
	var diags diag.Diagnostics
	diags = append(diags, checkCatalogs(u)...)
	diags = append(diags, checkSchemas(u)...)
	diags = append(diags, checkGrants(u)...)
	diags = append(diags, checkStorageCredentials(u)...)
	diags = append(diags, checkExternalLocations(u)...)
	diags = append(diags, checkVolumes(u)...)
	diags = append(diags, checkConnections(u)...)
	return diags
}

func checkCatalogs(u *ucm.Ucm) diag.Diagnostics {
	var diags diag.Diagnostics
	for _, key := range sortedKeys(u.Config.Resources.Catalogs) {
		c := u.Config.Resources.Catalogs[key]
		if c == nil || c.Name == "" {
			diags = append(diags, missingField(u, "catalogs", key, "name"))
		}
	}
	return diags
}

func checkSchemas(u *ucm.Ucm) diag.Diagnostics {
	var diags diag.Diagnostics
	for _, key := range sortedKeys(u.Config.Resources.Schemas) {
		s := u.Config.Resources.Schemas[key]
		if s == nil {
			continue
		}
		if s.Name == "" {
			diags = append(diags, missingField(u, "schemas", key, "name"))
		}
		if s.CatalogName == "" {
			diags = append(diags, missingField(u, "schemas", key, "catalog_name"))
		}
	}
	return diags
}

func checkGrants(u *ucm.Ucm) diag.Diagnostics {
	var diags diag.Diagnostics
	for _, key := range sortedKeys(u.Config.Resources.Grants) {
		g := u.Config.Resources.Grants[key]
		if g == nil {
			continue
		}
		if g.Principal == "" {
			diags = append(diags, missingField(u, "grants", key, "principal"))
		}
		if len(g.Privileges) == 0 {
			diags = append(diags, missingField(u, "grants", key, "privileges"))
		}
		if g.Securable.Type == "" {
			diags = append(diags, missingField(u, "grants", key, "securable.type"))
		}
		if g.Securable.Name == "" {
			diags = append(diags, missingField(u, "grants", key, "securable.name"))
		}
	}
	return diags
}

func checkStorageCredentials(u *ucm.Ucm) diag.Diagnostics {
	var diags diag.Diagnostics
	for _, key := range sortedKeys(u.Config.Resources.StorageCredentials) {
		c := u.Config.Resources.StorageCredentials[key]
		if c == nil {
			continue
		}
		if c.Name == "" {
			diags = append(diags, missingField(u, "storage_credentials", key, "name"))
		}
		set := 0
		if c.AwsIamRole != nil {
			set++
		}
		if c.AzureManagedIdentity != nil {
			set++
		}
		if c.AzureServicePrincipal != nil {
			set++
		}
		if c.DatabricksGcpServiceAccount != nil {
			set++
		}
		if set != 1 {
			path := resourcePath("storage_credentials", key)
			diags = append(diags, diag.Diagnostic{
				Severity: diag.Error,
				Summary: fmt.Sprintf(
					"storage_credential %q: exactly one of aws_iam_role, azure_managed_identity, azure_service_principal, databricks_gcp_service_account must be set (found %d)",
					key, set,
				),
				Paths:     []dyn.Path{path},
				Locations: locationsAt(u, path),
			})
		}
	}
	return diags
}

func checkExternalLocations(u *ucm.Ucm) diag.Diagnostics {
	var diags diag.Diagnostics
	for _, key := range sortedKeys(u.Config.Resources.ExternalLocations) {
		e := u.Config.Resources.ExternalLocations[key]
		if e == nil {
			continue
		}
		if e.Name == "" {
			diags = append(diags, missingField(u, "external_locations", key, "name"))
		}
		if e.Url == "" {
			diags = append(diags, missingField(u, "external_locations", key, "url"))
		}
		if e.CredentialName == "" {
			diags = append(diags, missingField(u, "external_locations", key, "credential_name"))
		}
	}
	return diags
}

func checkVolumes(u *ucm.Ucm) diag.Diagnostics {
	var diags diag.Diagnostics
	for _, key := range sortedKeys(u.Config.Resources.Volumes) {
		v := u.Config.Resources.Volumes[key]
		if v == nil {
			continue
		}
		if v.Name == "" {
			diags = append(diags, missingField(u, "volumes", key, "name"))
		}
		if v.CatalogName == "" {
			diags = append(diags, missingField(u, "volumes", key, "catalog_name"))
		}
		if v.SchemaName == "" {
			diags = append(diags, missingField(u, "volumes", key, "schema_name"))
		}
		if v.VolumeType == "" {
			diags = append(diags, missingField(u, "volumes", key, "volume_type"))
		}
	}
	return diags
}

func checkConnections(u *ucm.Ucm) diag.Diagnostics {
	var diags diag.Diagnostics
	for _, key := range sortedKeys(u.Config.Resources.Connections) {
		c := u.Config.Resources.Connections[key]
		if c == nil {
			continue
		}
		if c.Name == "" {
			diags = append(diags, missingField(u, "connections", key, "name"))
		}
		if c.ConnectionType == "" {
			diags = append(diags, missingField(u, "connections", key, "connection_type"))
		}
		if len(c.Options) == 0 {
			diags = append(diags, missingField(u, "connections", key, "options"))
		}
	}
	return diags
}

// missingField builds the standard "required field X is not set" diagnostic
// pointing at resources.<kind>.<key>.
func missingField(u *ucm.Ucm, kind, key, field string) diag.Diagnostic {
	resPath := resourcePath(kind, key)
	return diag.Diagnostic{
		Severity:  diag.Error,
		Summary:   fmt.Sprintf("%s %q: required field %q is not set", singularize(kind), key, field),
		Paths:     []dyn.Path{resPath},
		Locations: locationsAt(u, resPath),
	}
}

func resourcePath(kind, key string) dyn.Path {
	return dyn.NewPath(dyn.Key("resources"), dyn.Key(kind), dyn.Key(key))
}

func locationsAt(u *ucm.Ucm, p dyn.Path) []dyn.Location {
	v, err := dyn.GetByPath(u.Config.Value(), p)
	if err != nil {
		return nil
	}
	loc := v.Location()
	if loc.File == "" && loc.Line == 0 {
		return nil
	}
	return []dyn.Location{loc}
}

func sortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// singularize maps the plural map-field name used in YAML (e.g. "catalogs")
// to the singular noun used in diagnostics (e.g. "catalog").
func singularize(kind string) string {
	if s, ok := singular[kind]; ok {
		return s
	}
	return kind
}

var singular = map[string]string{
	"catalogs":            "catalog",
	"schemas":             "schema",
	"grants":              "grant",
	"storage_credentials": "storage_credential",
	"external_locations":  "external_location",
	"volumes":             "volume",
	"connections":         "connection",
}

// resourceKinds is the ordered list of kinds that validators iterate over.
// Kept here so new kinds only need to be added in one place.
var resourceKinds = []string{
	"catalogs",
	"schemas",
	"grants",
	"storage_credentials",
	"external_locations",
	"volumes",
	"connections",
}

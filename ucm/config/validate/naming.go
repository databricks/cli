package validate

import (
	"context"
	"fmt"
	"regexp"

	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/ucm"
)

// Naming enforces identifier rules on resource map keys and on UC names.
//
//   - Map keys under resources.<kind>.<key> must match [A-Za-z_][A-Za-z0-9_-]*
//     and be ≤128 chars. No dots, slashes, or whitespace — these break
//     ${resources.<kind>.<key>.<field>} interpolation.
//   - UC names (catalog.name, schema.name, etc.) must not contain slashes or
//     leading/trailing whitespace. Max 255 chars.
//
// Stricter UC identifier rules (reserved words, case handling) are a server-
// side concern and not replicated here.
func Naming() ucm.Mutator { return &naming{} }

type naming struct{}

func (m *naming) Name() string { return "validate:naming" }

const (
	maxKeyLen  = 128
	maxUCName  = 255
	keyPattern = `^[A-Za-z_][A-Za-z0-9_-]*$`
)

var keyRegex = regexp.MustCompile(keyPattern)

func (m *naming) Apply(_ context.Context, u *ucm.Ucm) diag.Diagnostics {
	var diags diag.Diagnostics
	for _, kind := range resourceKinds {
		for _, key := range resourceKeysOf(u, kind) {
			if d := validateKey(u, kind, key); d != nil {
				diags = append(diags, *d)
			}
			if d := validateUCName(u, kind, key); d != nil {
				diags = append(diags, *d)
			}
		}
	}
	return diags
}

func validateKey(u *ucm.Ucm, kind, key string) *diag.Diagnostic {
	if keyRegex.MatchString(key) && len(key) <= maxKeyLen {
		return nil
	}
	p := resourcePath(kind, key)
	return &diag.Diagnostic{
		Severity: diag.Error,
		Summary: fmt.Sprintf(
			"resource key %q under resources.%s is invalid (must match %s, max %d chars)",
			key, kind, keyPattern, maxKeyLen,
		),
		Paths:     []dyn.Path{p},
		Locations: locationsAt(u, p),
	}
}

// validateUCName checks the `name` field (and other UC-name-bearing fields)
// for slash characters or overlong values. Grants carry no `name` field; they
// are skipped.
func validateUCName(u *ucm.Ucm, kind, key string) *diag.Diagnostic {
	name, field, ok := ucNameOf(u, kind, key)
	if !ok || name == "" {
		return nil
	}
	if !containsForbidden(name) && len(name) <= maxUCName {
		return nil
	}
	p := resourcePath(kind, key).Append(dyn.Key(field))
	return &diag.Diagnostic{
		Severity: diag.Error,
		Summary: fmt.Sprintf(
			"%s %q: %s=%q contains forbidden characters or exceeds %d chars",
			singularize(kind), key, field, name, maxUCName,
		),
		Paths:     []dyn.Path{p},
		Locations: locationsAt(u, p),
	}
}

// containsForbidden reports whether s has any character that UC rejects in
// identifier position (slashes, backticks, whitespace).
func containsForbidden(s string) bool {
	for _, r := range s {
		switch r {
		case '/', '\\', '`', ' ', '\t', '\n', '\r':
			return true
		}
	}
	return false
}

// ucNameOf returns the UC-identifier-bearing field for a given resource
// (kind, key), plus the JSON field name so diagnostics can point at the
// right subfield.
func ucNameOf(u *ucm.Ucm, kind, key string) (string, string, bool) {
	switch kind {
	case "catalogs":
		if c := u.Config.Resources.Catalogs[key]; c != nil {
			return c.Name, "name", true
		}
	case "schemas":
		if s := u.Config.Resources.Schemas[key]; s != nil {
			return s.Name, "name", true
		}
	case "storage_credentials":
		if c := u.Config.Resources.StorageCredentials[key]; c != nil {
			return c.Name, "name", true
		}
	case "external_locations":
		if e := u.Config.Resources.ExternalLocations[key]; e != nil {
			return e.Name, "name", true
		}
	case "volumes":
		if v := u.Config.Resources.Volumes[key]; v != nil {
			return v.Name, "name", true
		}
	case "connections":
		if c := u.Config.Resources.Connections[key]; c != nil {
			return c.Name, "name", true
		}
	}
	return "", "", false
}

func resourceKeysOf(u *ucm.Ucm, kind string) []string {
	switch kind {
	case "catalogs":
		return sortedKeys(u.Config.Resources.Catalogs)
	case "schemas":
		return sortedKeys(u.Config.Resources.Schemas)
	case "grants":
		return sortedKeys(u.Config.Resources.Grants)
	case "storage_credentials":
		return sortedKeys(u.Config.Resources.StorageCredentials)
	case "external_locations":
		return sortedKeys(u.Config.Resources.ExternalLocations)
	case "volumes":
		return sortedKeys(u.Config.Resources.Volumes)
	case "connections":
		return sortedKeys(u.Config.Resources.Connections)
	}
	return nil
}

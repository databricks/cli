package validate

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"unicode"
	"unicode/utf8"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/internal/validation/generated"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/dyn"
)

type compiledIdentifierRules struct {
	trie   *dyn.TrieNode
	fields map[string][]string
}

// identifierFields explicitly defines identifier semantics instead of assuming every
// field ending in "_name" is an identifier. True means omission is also an error.
var identifierFields = map[string]bool{
	"catalog_name":           false,
	"credential_name":        false,
	"database_instance_name": false,
	"database_name":          false,
	"display_name":           true,
	"endpoint_name":          false,
	"instance_pool_name":     true,
	"name":                   true,
	"output_schema_name":     false,
	"schema_name":            false,
	"table_name":             false,
}

// supplementalIdentifierFields covers backend requirements absent from OpenAPI.
var supplementalIdentifierFields = map[string][]string{
	"resources.dashboards.*":        {"display_name"},
	"resources.registered_models.*": {"catalog_name", "name", "schema_name"},
	"resources.sql_warehouses.*":    {"name"},
}

var identifierRules = sync.OnceValues(compileIdentifierRules)

func compileIdentifierRules() (compiledIdentifierRules, error) {
	fields := make(map[string][]string)
	for pattern, required := range generated.RequiredFields {
		if !isIdentifierObjectPattern(pattern) {
			continue
		}
		for _, field := range required {
			if _, ok := identifierFields[field]; ok {
				fields[pattern] = append(fields[pattern], field)
			}
		}
	}
	for pattern, supplemental := range supplementalIdentifierFields {
		fields[pattern] = append(fields[pattern], supplemental...)
	}

	trie := &dyn.TrieNode{}
	for value := range fields {
		pattern, err := dyn.NewPatternFromString(value)
		if err != nil {
			return compiledIdentifierRules{}, fmt.Errorf("invalid pattern %q for identifier validation: %w", value, err)
		}
		if err := trie.Insert(pattern); err != nil {
			return compiledIdentifierRules{}, fmt.Errorf("failed to insert pattern %q into trie: %w", value, err)
		}
	}
	return compiledIdentifierRules{trie: trie, fields: fields}, nil
}

// validateIdentifiers rejects values that the backend cannot use as identifiers.
func validateIdentifiers(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	rules, err := identifierRules()
	if err != nil {
		return diag.FromErr(err)
	}

	var diags diag.Diagnostics
	err = dyn.WalkReadOnly(b.Config.Value(), func(path dyn.Path, value dyn.Value) error {
		pattern, ok := rules.trie.SearchPath(path)
		if !ok {
			return nil
		}
		for _, name := range rules.fields[pattern.String()] {
			field := value.Get(name)
			if field.Kind() == dyn.KindInvalid || field.Kind() == dyn.KindNil {
				if identifierFields[name] {
					diags = diags.Append(identifierDiag(b, path, name, "is required", ""))
				}
				continue
			}
			if field.Kind() != dyn.KindString {
				continue
			}
			reason, detail := invalidIdentifierReason(field.MustString())
			if reason != "" {
				diags = diags.Append(identifierDiag(b, path, name, reason, detail))
			}
		}
		return nil
	})
	if err != nil {
		return diag.FromErr(err)
	}
	return diags
}

func missingIdentifierIsError(pattern, field string) bool {
	return isIdentifierObjectPattern(pattern) && identifierFields[field]
}

func isIdentifierObjectPattern(pattern string) bool {
	if pattern == "bundle" {
		return true
	}
	parts := strings.Split(pattern, ".")
	return len(parts) == 3 && parts[0] == "resources" && parts[2] == "*"
}

func identifierDiag(b *bundle.Bundle, resourcePath dyn.Path, field, reason, detail string) diag.Diagnostic {
	fieldPath := resourcePath.Append(dyn.Key(field))
	return diag.Diagnostic{
		Severity:  diag.Error,
		Summary:   fmt.Sprintf("%s %s %s", requiredObjectName(resourcePath), field, reason),
		Detail:    detail,
		Locations: locationsFor(b, fieldPath, resourcePath),
		Paths:     []dyn.Path{fieldPath},
	}
}

func locationsFor(b *bundle.Bundle, path, fallback dyn.Path) []dyn.Location {
	if v, err := dyn.GetByPath(b.Config.Value(), path); err == nil && len(v.Locations()) > 0 {
		return v.Locations()
	}
	v, err := dyn.GetByPath(b.Config.Value(), fallback)
	if err != nil {
		return nil
	}
	return v.Locations()
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

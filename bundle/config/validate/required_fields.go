package validate

import (
	"context"
	"fmt"
	"maps"
	"slices"
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

type missingFieldBehavior int

const (
	warnIfMissing missingFieldBehavior = iota
	errorIfMissing
	ignoreIfMissing
)

type fieldRule struct {
	name               string
	missing            missingFieldBehavior
	validateIdentifier bool
}

type compiledFieldRules struct {
	trie   *dyn.TrieNode
	fields map[string][]fieldRule
}

var requiredFieldRules = sync.OnceValues(compileRequiredFieldRules)

// supplementalFieldRules covers backend requirements missing from the OpenAPI spec.
var supplementalFieldRules = map[string][]fieldRule{
	"resources.dashboards.*": {
		{name: "display_name", missing: errorIfMissing, validateIdentifier: true},
		{name: "warehouse_id", missing: errorIfMissing},
	},
	"resources.registered_models.*": {
		{name: "name", missing: errorIfMissing, validateIdentifier: true},
		{name: "catalog_name", missing: ignoreIfMissing, validateIdentifier: true},
		{name: "schema_name", missing: ignoreIfMissing, validateIdentifier: true},
	},
	"resources.sql_warehouses.*": {
		{name: "name", missing: errorIfMissing, validateIdentifier: true},
	},
}

// compileRequiredFieldRules combines generated and backend-specific field requirements.
func compileRequiredFieldRules() (compiledFieldRules, error) {
	fields := make(map[string][]fieldRule, len(generated.RequiredFields)+len(supplementalFieldRules))
	for pattern, required := range generated.RequiredFields {
		rules := make([]fieldRule, 0, len(required))
		for _, field := range required {
			rule := fieldRule{name: field, missing: warnIfMissing}
			if isResourcePathPattern(pattern) && isIdentifierField(field) {
				rule.validateIdentifier = true
				if isResourceNameField(field) {
					rule.missing = errorIfMissing
				}
			}
			if pattern == "bundle" && field == "name" {
				rule.validateIdentifier = true
				rule.missing = errorIfMissing
			}
			rules = append(rules, rule)
		}
		fields[pattern] = rules
	}
	maps.Copy(fields, supplementalFieldRules)

	trie := &dyn.TrieNode{}
	for value := range fields {
		pattern, err := dyn.NewPatternFromString(value)
		if err != nil {
			return compiledFieldRules{}, fmt.Errorf("invalid pattern %q for required field validation: %w", value, err)
		}
		if err := trie.Insert(pattern); err != nil {
			return compiledFieldRules{}, fmt.Errorf("failed to insert pattern %q into trie: %w", value, err)
		}
	}
	return compiledFieldRules{trie: trie, fields: fields}, nil
}

// validateRequiredFields checks generated OpenAPI requirements and supplemental
// backend requirements in one walk of the resolved configuration.
func validateRequiredFields(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	rules, err := requiredFieldRules()
	if err != nil {
		return diag.FromErr(err)
	}

	var diags diag.Diagnostics
	err = dyn.WalkReadOnly(b.Config.Value(), func(path dyn.Path, value dyn.Value) error {
		pattern, ok := rules.trie.SearchPath(path)
		if !ok {
			return nil
		}
		for _, rule := range rules.fields[pattern.String()] {
			field := value.Get(rule.name)
			if field.Kind() == dyn.KindInvalid || field.Kind() == dyn.KindNil {
				switch rule.missing {
				case errorIfMissing:
					diags = diags.Append(identifierOrRequiredFieldDiag(b, path, rule, "is required", ""))
				case warnIfMissing:
					diags = diags.Append(missingFieldWarning(path, value, rule.name))
				case ignoreIfMissing:
					continue
				}
				continue
			}
			if field.Kind() != dyn.KindString || !rule.validateIdentifier {
				continue
			}
			reason, detail := invalidIdentifierReason(field.MustString())
			if reason != "" {
				diags = diags.Append(identifierOrRequiredFieldDiag(b, path, rule, reason, detail))
			}
		}
		return nil
	})
	if err != nil {
		return diag.FromErr(err)
	}
	return diags
}

// identifierOrRequiredFieldDiag reports an invalid or absent required field.
func identifierOrRequiredFieldDiag(b *bundle.Bundle, resourcePath dyn.Path, rule fieldRule, reason, detail string) diag.Diagnostic {
	fieldPath := resourcePath.Append(dyn.Key(rule.name))
	entity := requiredObjectName(resourcePath)
	summary := fmt.Sprintf("%s %s %s", entity, rule.name, reason)
	return diag.Diagnostic{
		Severity:  diag.Error,
		Summary:   summary,
		Detail:    detail,
		Locations: locationsFor(b, fieldPath, resourcePath),
		Paths:     []dyn.Path{fieldPath},
	}
}

// missingFieldWarning preserves the warning behavior for ordinary OpenAPI requirements.
func missingFieldWarning(path dyn.Path, value dyn.Value, field string) diag.Diagnostic {
	return diag.Diagnostic{
		Severity:  diag.Warning,
		Summary:   fmt.Sprintf("required field %q is not set", field),
		Locations: value.Locations(),
		Paths:     []dyn.Path{slices.Clone(path)},
	}
}

// locationsFor resolves the location of path, falling back to fallback when the field
// carries none of its own (an omitted field has no location to point at).
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

// invalidIdentifierReason explains why a resource-level identifier is invalid.
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

// isIdentifierField reports whether a field carries a resource identifier or reference.
func isIdentifierField(field string) bool {
	return field == "name" || strings.HasSuffix(field, "_name")
}

// isResourceNameField reports whether a field names the resource itself.
func isResourceNameField(field string) bool {
	switch field {
	case "name", "display_name", "instance_pool_name":
		return true
	default:
		return false
	}
}

// isResourcePathPattern reports whether a pattern selects top-level bundle resources.
func isResourcePathPattern(pattern string) bool {
	parts := strings.Split(pattern, ".")
	return len(parts) == 3 && parts[0] == "resources" && parts[2] == "*"
}

// requiredObjectName returns the object name used in a required-field diagnostic.
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

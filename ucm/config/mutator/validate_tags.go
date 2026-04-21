package mutator

import (
	"context"
	"fmt"
	"slices"
	"sort"

	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/ucm"
	"github.com/databricks/cli/ucm/config/resources"
)

type validateTags struct{}

// ValidateTags enforces every resources.tag_validation_rules.* entry against
// every matching securable in the resources tree.
//
// For each rule + matching securable:
//   - every key in Required must be present on the securable's `tags` map
//   - if the rule sets AllowedValues for a key, the tag value (when present)
//     must be a member
//
// Emits error-level diagnostics with source-location info pointing at the
// offending securable, so editor integrations can jump to the right spot.
//
// No dependency on UC's server-side tag policy — this is ucm's own gate,
// run during `validate`, `plan`, and `policy-check`.
func ValidateTags() ucm.Mutator { return &validateTags{} }

func (m *validateTags) Name() string { return "ValidateTags" }

// securableFieldsToKind maps the Resources struct field (as it appears under
// `resources:` in ucm.yml) to the rule's `securable_types` token.
var securableFieldsToKind = map[string]string{
	"catalogs": "catalog",
	"schemas":  "schema",
}

func (m *validateTags) Apply(_ context.Context, u *ucm.Ucm) diag.Diagnostics {
	rules := u.Config.Resources.TagValidationRules
	if len(rules) == 0 {
		return nil
	}

	ruleNames := sortedKeys(rules)

	var diags diag.Diagnostics
	for _, field := range []string{"catalogs", "schemas"} {
		kind := securableFieldsToKind[field]
		for _, resourceName := range securableNames(u, field) {
			tags := securableTags(u, field, resourceName)
			for _, ruleName := range ruleNames {
				rule := rules[ruleName]
				if rule == nil || !slices.Contains(rule.SecurableTypes, kind) {
					continue
				}
				diags = append(diags, evaluateRule(u, field, resourceName, tags, ruleName, rule)...)
			}
		}
	}
	return diags
}

func evaluateRule(
	u *ucm.Ucm,
	field, resourceName string,
	tags map[string]string,
	ruleName string,
	rule *resources.TagValidationRule,
) diag.Diagnostics {
	var diags diag.Diagnostics

	tagsPath := dyn.NewPath(
		dyn.Key("resources"),
		dyn.Key(field),
		dyn.Key(resourceName),
		dyn.Key("tags"),
	)

	for _, key := range sortedStrings(rule.Required) {
		if _, ok := tags[key]; !ok {
			diags = append(diags, diag.Diagnostic{
				Severity: diag.Error,
				Summary: fmt.Sprintf(
					"tag-validation-rule %q requires tag %q on %s %q",
					ruleName, key, securableFieldsToKind[field], resourceName,
				),
				Paths:     []dyn.Path{tagsPath},
				Locations: locations(u, tagsPath),
			})
		}
	}

	for _, key := range sortedKeys(rule.AllowedValues) {
		allowed := rule.AllowedValues[key]
		value, ok := tags[key]
		if !ok {
			continue
		}
		if !slices.Contains(allowed, value) {
			diags = append(diags, diag.Diagnostic{
				Severity: diag.Error,
				Summary: fmt.Sprintf(
					"tag-validation-rule %q: %s %q tag %q=%q is not in allowed values %v",
					ruleName, securableFieldsToKind[field], resourceName, key, value, allowed,
				),
				Paths:     []dyn.Path{tagsPath.Append(dyn.Key(key))},
				Locations: locations(u, tagsPath.Append(dyn.Key(key))),
			})
		}
	}

	return diags
}

func securableNames(u *ucm.Ucm, field string) []string {
	switch field {
	case "catalogs":
		return sortedKeys(u.Config.Resources.Catalogs)
	case "schemas":
		return sortedKeys(u.Config.Resources.Schemas)
	}
	return nil
}

func securableTags(u *ucm.Ucm, field, name string) map[string]string {
	switch field {
	case "catalogs":
		if c := u.Config.Resources.Catalogs[name]; c != nil {
			return c.Tags
		}
	case "schemas":
		if s := u.Config.Resources.Schemas[name]; s != nil {
			return s.Tags
		}
	}
	return nil
}

func locations(u *ucm.Ucm, p dyn.Path) []dyn.Location {
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

func sortedStrings(in []string) []string {
	out := slices.Clone(in)
	sort.Strings(out)
	return out
}

func sortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

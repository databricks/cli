package validate

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

func (m *validateTags) Name() string { return "validate:tags" }

func (m *validateTags) Apply(_ context.Context, u *ucm.Ucm) diag.Diagnostics {
	rules := u.Config.Resources.TagValidationRules
	if len(rules) == 0 {
		return nil
	}

	ruleNames := sortedKeys(rules)

	var diags diag.Diagnostics
	for _, kind := range []string{"catalogs", "schemas"} {
		singular := singularize(kind)
		for _, resourceName := range securableNames(u, kind) {
			tags := securableTags(u, kind, resourceName)
			for _, ruleName := range ruleNames {
				rule := rules[ruleName]
				if rule == nil || !slices.Contains(rule.SecurableTypes, singular) {
					continue
				}
				diags = append(diags, evaluateRule(u, kind, resourceName, tags, ruleName, rule)...)
			}
		}
	}
	return diags
}

func evaluateRule(
	u *ucm.Ucm,
	kind, resourceName string,
	tags map[string]string,
	ruleName string,
	rule *resources.TagValidationRule,
) diag.Diagnostics {
	var diags diag.Diagnostics

	tagsPath := resourcePath(kind, resourceName).Append(dyn.Key("tags"))
	singular := singularize(kind)

	for _, key := range sortedStrings(rule.Required) {
		if _, ok := tags[key]; !ok {
			diags = append(diags, diag.Diagnostic{
				Severity: diag.Error,
				Summary: fmt.Sprintf(
					"tag-validation-rule %q requires tag %q on %s %q",
					ruleName, key, singular, resourceName,
				),
				Paths:     []dyn.Path{tagsPath},
				Locations: locationsAt(u, tagsPath),
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
			keyPath := tagsPath.Append(dyn.Key(key))
			diags = append(diags, diag.Diagnostic{
				Severity: diag.Error,
				Summary: fmt.Sprintf(
					"tag-validation-rule %q: %s %q tag %q=%q is not in allowed values %v",
					ruleName, singular, resourceName, key, value, allowed,
				),
				Paths:     []dyn.Path{keyPath},
				Locations: locationsAt(u, keyPath),
			})
		}
	}

	return diags
}

func securableNames(u *ucm.Ucm, kind string) []string {
	switch kind {
	case "catalogs":
		return sortedKeys(u.Config.Resources.Catalogs)
	case "schemas":
		return sortedKeys(u.Config.Resources.Schemas)
	}
	return nil
}

func securableTags(u *ucm.Ucm, kind, name string) map[string]string {
	switch kind {
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

func sortedStrings(in []string) []string {
	out := slices.Clone(in)
	sort.Strings(out)
	return out
}

package dresources

import (
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.yaml.in/yaml/v3"
)

// normalizeConfig projects a ResourceLifecycleConfig into an order-independent
// "category -> sorted rules" shape so two configs can be compared for behavioural
// equivalence regardless of the order fields were authored in.
func normalizeConfig(c ResourceLifecycleConfig) map[string][]string {
	fieldRules := func(rules []FieldRule) []string {
		out := make([]string, 0, len(rules))
		for _, r := range rules {
			out = append(out, r.Field.String()+"|"+r.Reason)
		}
		slices.Sort(out)
		return out
	}

	backend := make([]string, 0, len(c.BackendDefaults))
	for _, r := range c.BackendDefaults {
		backend = append(backend, r.Field.String())
	}
	slices.Sort(backend)

	hashed := slices.Clone(c.HashedFields)
	slices.Sort(hashed)

	return map[string][]string{
		"ignore_remote_changes": fieldRules(c.IgnoreRemoteChanges),
		"ignore_local_changes":  fieldRules(c.IgnoreLocalChanges),
		"recreate_on_changes":   fieldRules(c.RecreateOnChanges),
		"provided_id_fields":    fieldRules(c.ProvidedIDFields),
		"updatable_id_fields":   fieldRules(c.UpdatableIDFields),
		"normalize_slash":       fieldRules(c.NormalizeSlash),
		"sensitive_fields":      fieldRules(c.SensitiveFields),
		"stable_output_fields":  fieldRules(c.StableOutputFields),
		"backend_defaults":      backend,
		"hashed_fields":         hashed,
	}
}

// TestFieldPolicyLowersToLiveConfig proves the field-keyed configs_v2/<r>.yml files
// lower to exactly the same lifecycle behaviour as the flat hand-written configs/<r>.yml
// they replace. As long as this passes, the plan ladder and every other consumer are
// unaffected, so the migration is behaviour-preserving.
func TestFieldPolicyLowersToLiveConfig(t *testing.T) {
	migrated := []string{
		"volumes", "schemas", "catalogs", "experiments",
		"postgres_roles", "postgres_projects", "postgres_branches",
		"dashboards", "secrets",
	}
	for _, resourceType := range migrated {
		t.Run(resourceType, func(t *testing.T) {
			v2 := GetFieldPolicyConfig(resourceType)
			require.NotNil(t, v2, "no configs_v2/%s.yml", resourceType)
			live := GetResourceConfig(resourceType)
			assert.Equal(t, normalizeConfig(*live), normalizeConfig(*v2))
		})
	}
}

// TestFieldPolicyRejectsDuplicateField shows the redundancy class the flat format
// needed a dedicated exclusivity test for is now unrepresentable: a field can carry
// only one policy, and a repeated key is a load error, not a silently-shadowed rule.
func TestFieldPolicyRejectsDuplicateField(t *testing.T) {
	var fpc FieldPolicyConfig
	err := yaml.Unmarshal([]byte("fields:\n  name: { action: id }\n  name: { action: immutable }\n"), &fpc)
	require.Error(t, err)
	assert.Contains(t, err.Error(), `already defined`)
}

// TestFieldPolicyRejectsUnknownAction proves the action enum is enforced by the
// parser: an unknown value fails at yaml.Unmarshal, not later in Lower.
func TestFieldPolicyRejectsUnknownAction(t *testing.T) {
	var fpc FieldPolicyConfig
	err := yaml.Unmarshal([]byte("fields:\n  name: { action: teleport }\n"), &fpc)
	require.Error(t, err)
	assert.Contains(t, err.Error(), `unknown action "teleport"`)
}

// TestFieldPolicyEveryActionLowers keeps validActions and the lowerAction switch in
// sync: every value the parser accepts must lower without hitting the default guard.
func TestFieldPolicyEveryActionLowers(t *testing.T) {
	for a := range validActions {
		_, err := FieldPolicyConfig{Fields: map[string]FieldPolicy{"name": {Action: a}}}.Lower()
		require.NoErrorf(t, err, "action %q", a)
	}
}

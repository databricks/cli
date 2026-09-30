package dresources

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.yaml.in/yaml/v3"
)

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

// TestFieldPolicyLowerVolumes pins the lowering of a representative resource that
// exercises id, id_renameable, immutable, and the trim_slash comparison modifier.
func TestFieldPolicyLowerVolumes(t *testing.T) {
	rules := func(rs []FieldRule) []string {
		var out []string
		for _, r := range rs {
			out = append(out, r.Field.String()+"|"+r.Reason)
		}
		return out
	}
	c := GetResourceConfig("volumes")
	assert.ElementsMatch(t, []string{"catalog_name|id_field", "schema_name|id_field"}, rules(c.ProvidedIDFields))
	assert.ElementsMatch(t, []string{"name|id_changes"}, rules(c.UpdatableIDFields))
	assert.ElementsMatch(t, []string{"storage_location|immutable", "volume_type|immutable"}, rules(c.RecreateOnChanges))
	assert.ElementsMatch(t, []string{"storage_location|uc_strips_trailing_slash"}, rules(c.NormalizeSlash))
}

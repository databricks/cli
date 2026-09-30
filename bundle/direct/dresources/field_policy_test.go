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
	err := yaml.Unmarshal([]byte("fields:\n  name: [id]\n  name: [immutable]\n"), &fpc)
	require.Error(t, err)
	assert.Contains(t, err.Error(), `already defined`)
}

// TestFieldPolicyRejectsUnknownToken proves the behaviour vocabulary is enforced by the parser.
func TestFieldPolicyRejectsUnknownToken(t *testing.T) {
	var fpc FieldPolicyConfig
	err := yaml.Unmarshal([]byte("fields:\n  name: [teleport]\n"), &fpc)
	require.Error(t, err)
	assert.Contains(t, err.Error(), `unknown behaviour "teleport"`)
}

func rulesOf(rs []FieldRule) []string {
	var out []string
	for _, r := range rs {
		out = append(out, r.Field.String()+"|"+r.Reason)
	}
	return out
}

// TestFieldPolicyLowerVolumes pins the list-form lowering of a representative resource that
// exercises id, renameable_id, immutable, and the trim_slash comparison modifier.
func TestFieldPolicyLowerVolumes(t *testing.T) {
	c := GetResourceConfig("volumes")
	assert.ElementsMatch(t, []string{"catalog_name|id_field", "schema_name|id_field"}, rulesOf(c.ProvidedIDFields))
	assert.ElementsMatch(t, []string{"name|id_changes"}, rulesOf(c.UpdatableIDFields))
	assert.ElementsMatch(t, []string{"storage_location|immutable", "volume_type|immutable"}, rulesOf(c.RecreateOnChanges))
	assert.ElementsMatch(t, []string{"storage_location|uc_strips_trailing_slash"}, rulesOf(c.NormalizeSlash))
}

// TestFieldPolicyTokensLower checks each behaviour token maps to the right flat rule, and
// that spec-derived reasons pick up the "spec:" prefix only in generated configs.
func TestFieldPolicyTokensLower(t *testing.T) {
	parse := func(yml string, generated bool) ResourceLifecycleConfig {
		var fpc FieldPolicyConfig
		require.NoError(t, yaml.Unmarshal([]byte(yml), &fpc))
		rc, err := fpc.Lower(generated)
		require.NoError(t, err)
		return rc
	}

	hand := parse("fields:\n  a: [immutable, input_only]\n  b: [mutable_output]\n  c: [immutable_output]\n", false)
	assert.ElementsMatch(t, []string{"a|immutable"}, rulesOf(hand.RecreateOnChanges))
	assert.ElementsMatch(t, []string{"a|input_only", "b|output_only"}, rulesOf(hand.IgnoreRemoteChanges))
	assert.ElementsMatch(t, []string{"c|"}, rulesOf(hand.StableOutputFields))

	gen := parse("fields:\n  a: [immutable, input_only]\n  b: [mutable_output]\n", true)
	assert.ElementsMatch(t, []string{"a|spec:immutable"}, rulesOf(gen.RecreateOnChanges))
	assert.ElementsMatch(t, []string{"a|spec:input_only", "b|spec:output_only"}, rulesOf(gen.IgnoreRemoteChanges))
}

// TestFieldPolicyMapForm checks the map form for value-bearing behaviours.
func TestFieldPolicyMapForm(t *testing.T) {
	var fpc FieldPolicyConfig
	require.NoError(t, yaml.Unmarshal([]byte("fields:\n  v: { ignore_remote: etag_based, hashed: true }\n  r: { ignore_local: input_only }\n"), &fpc))
	rc, err := fpc.Lower(false)
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"v|etag_based"}, rulesOf(rc.IgnoreRemoteChanges))
	assert.ElementsMatch(t, []string{"r|input_only"}, rulesOf(rc.IgnoreLocalChanges))
	assert.Equal(t, []string{"v"}, rc.HashedFields)
}

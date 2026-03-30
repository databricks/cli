package mutator

import (
	"math"
	"reflect"
	"testing"

	"github.com/databricks/cli/libs/dyn"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSuggestionThreshold(t *testing.T) {
	tests := []struct {
		keyLen int
		want   int
	}{
		{0, 1}, // max(1, 0) = 1
		{1, 1}, // max(1, 0) = 1
		{2, 1}, // max(1, 1) = 1
		{3, 1}, // max(1, 1) = 1
		{4, 2},
		{5, 2},
		{6, 3},
		{7, 3},
		{100, 3}, // capped at 3
	}
	for _, tt := range tests {
		assert.Equal(t, tt.want, suggestionThreshold(tt.keyLen), "keyLen=%d", tt.keyLen)
	}
}

func TestClosestMatch(t *testing.T) {
	candidates := []string{"bundle", "workspace", "variables", "resources"}

	tests := []struct {
		key      string
		wantName string
		wantDist int
	}{
		{"bundle", "bundle", 0},
		{"bundl", "bundle", 1},
		{"bunlde", "bundle", 2},
		{"xxxxxxx", "", math.MaxInt}, // too far from any candidate
		{"var", "", math.MaxInt},     // distance 7+ from all candidates, too far
	}

	for _, tt := range tests {
		name, dist := closestMatch(tt.key, candidates)
		assert.Equal(t, tt.wantName, name, "key=%q", tt.key)
		assert.Equal(t, tt.wantDist, dist, "key=%q", tt.key)
	}
}

func TestClosestMatchEmptyCandidates(t *testing.T) {
	name, dist := closestMatch("foo", nil)
	assert.Equal(t, "", name)
	assert.Equal(t, math.MaxInt, dist)
}

func TestStructFieldNames(t *testing.T) {
	type Embedded struct {
		EmbeddedField string `json:"embedded_field"`
	}

	type TestStruct struct {
		Embedded
		Name     string `json:"name"`
		ID       string `json:"id,omitempty" bundle:"readonly"`
		Internal string `json:"internal_field" bundle:"internal"`
		Skipped  string `json:"-"`
		NoTag    string
	}

	names := structFieldNames(reflect.TypeOf(TestStruct{}))

	assert.Contains(t, names, "embedded_field")
	assert.Contains(t, names, "name")
	assert.Contains(t, names, "id")                // readonly should be included
	assert.NotContains(t, names, "internal_field") // internal should be excluded
	assert.NotContains(t, names, "-")
	assert.NotContains(t, names, "NoTag") // No json tag → excluded
}

func TestStructFieldNamesNonStruct(t *testing.T) {
	assert.Nil(t, structFieldNames(reflect.TypeOf("")))
	assert.Nil(t, structFieldNames(reflect.TypeOf(42)))
}

func TestMapKeysFromDyn(t *testing.T) {
	v := dyn.V(map[string]dyn.Value{
		"alpha": dyn.V("a"),
		"beta":  dyn.V("b"),
		"gamma": dyn.V("c"),
	})
	keys := mapKeysFromDyn(v)
	assert.ElementsMatch(t, []string{"alpha", "beta", "gamma"}, keys)
}

func TestMapKeysFromDynInvalid(t *testing.T) {
	assert.Nil(t, mapKeysFromDyn(dyn.InvalidValue))
	assert.Nil(t, mapKeysFromDyn(dyn.V("not a map")))
}

func TestRewriteToVarShorthand(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{"variables.my_cluster.value", "var.my_cluster"},
		{"variables.x.value", "var.x"},
		{"bundle.name", "bundle.name"},                         // not a variables path
		{"variables.foo.bar.value", "variables.foo.bar.value"}, // nested - don't rewrite
		{"variables.foo.default", "variables.foo.default"},     // not .value suffix
	}
	for _, tt := range tests {
		assert.Equal(t, tt.want, rewriteToVarShorthand(tt.in), "in=%q", tt.in)
	}
}

func TestSuggestPathSingleSegmentFix(t *testing.T) {
	// Simulate: ${bundle.nme} where "name" is the correct field.
	normalized := dyn.V(map[string]dyn.Value{
		"bundle": dyn.V(map[string]dyn.Value{
			"name": dyn.V("test"),
		}),
	})

	result := suggestPath([]string{"bundle", "nme"}, normalized)
	assert.Equal(t, "bundle.name", result)
}

func TestSuggestPathMultiSegmentFix(t *testing.T) {
	// Simulate: ${bundl.nme} where both segments are wrong.
	normalized := dyn.V(map[string]dyn.Value{
		"bundle": dyn.V(map[string]dyn.Value{
			"name": dyn.V("test"),
		}),
	})

	result := suggestPath([]string{"bundl", "nme"}, normalized)
	assert.Equal(t, "bundle.name", result)
}

func TestSuggestPathAllCorrect(t *testing.T) {
	normalized := dyn.V(map[string]dyn.Value{
		"bundle": dyn.V(map[string]dyn.Value{
			"name": dyn.V("test"),
		}),
	})

	// All segments match exactly → no suggestion needed.
	result := suggestPath([]string{"bundle", "name"}, normalized)
	assert.Equal(t, "", result)
}

func TestSuggestPathNoMatch(t *testing.T) {
	normalized := dyn.V(map[string]dyn.Value{
		"bundle": dyn.V(map[string]dyn.Value{
			"name": dyn.V("test"),
		}),
	})

	// "zzzzz" is too far from any candidate.
	result := suggestPath([]string{"bundle", "zzzzz"}, normalized)
	assert.Equal(t, "", result)
}

func TestSuggestPathMapKey(t *testing.T) {
	// Simulate: ${variables.my_clster.value} where "my_cluster" is the correct key.
	normalized := dyn.V(map[string]dyn.Value{
		"variables": dyn.V(map[string]dyn.Value{
			"my_cluster": dyn.V(map[string]dyn.Value{
				"value": dyn.V("abc"),
			}),
		}),
	})

	result := suggestPath([]string{"variables", "my_clster", "value"}, normalized)
	assert.Equal(t, "variables.my_cluster.value", result)
}

func TestSuggestPathResourceField(t *testing.T) {
	// The suggestion should work based on Go type information, even if the
	// dyn value doesn't have the field. We test that structFieldNames includes
	// readonly fields like "id".
	normalized := dyn.V(map[string]dyn.Value{
		"resources": dyn.V(map[string]dyn.Value{
			"jobs": dyn.V(map[string]dyn.Value{
				"my_job": dyn.V(map[string]dyn.Value{
					"name": dyn.V("test-job"),
				}),
			}),
		}),
	})

	result := suggestPath([]string{"resources", "jobs", "my_job", "nme"}, normalized)
	assert.Equal(t, "resources.jobs.my_job.name", result)
}

func TestSuggestVarRewriting(t *testing.T) {
	normalized := dyn.V(map[string]dyn.Value{
		"variables": dyn.V(map[string]dyn.Value{
			"my_cluster_id": dyn.V(map[string]dyn.Value{
				"value": dyn.V("abc-123"),
			}),
		}),
	})

	m := &resolveVariableReferences{
		prefixes: defaultPrefixes,
	}

	prefixes := []dyn.Path{dyn.MustPathFromString("variables")}
	varPath := dyn.NewPath(dyn.Key("var"))

	suggestion := m.suggest("var.my_clster_id", normalized, prefixes, varPath)
	require.Equal(t, "var.my_cluster_id", suggestion)
}

func TestSuggestVarPrefixTypo(t *testing.T) {
	normalized := dyn.V(map[string]dyn.Value{
		"variables": dyn.V(map[string]dyn.Value{
			"my_cluster_id": dyn.V(map[string]dyn.Value{
				"value": dyn.V("abc-123"),
			}),
		}),
	})

	m := &resolveVariableReferences{
		prefixes: defaultPrefixes,
	}

	prefixes := []dyn.Path{dyn.MustPathFromString("variables")}
	varPath := dyn.NewPath(dyn.Key("var"))

	// Typo in var prefix only, variable name is correct.
	assert.Equal(t, "var.my_cluster_id", m.suggest("vr.my_cluster_id", normalized, prefixes, varPath))

	// Typo in var prefix AND variable name.
	assert.Equal(t, "var.my_cluster_id", m.suggest("vr.my_clster_id", normalized, prefixes, varPath))

	// Var prefix typo but no matching variable.
	assert.Equal(t, "", m.suggest("vr.nonexistent", normalized, prefixes, varPath))
}

func TestSuggestNoSuggestionForCorrectPath(t *testing.T) {
	normalized := dyn.V(map[string]dyn.Value{
		"variables": dyn.V(map[string]dyn.Value{
			"my_cluster_id": dyn.V(map[string]dyn.Value{
				"value": dyn.V("abc-123"),
			}),
		}),
	})

	m := &resolveVariableReferences{
		prefixes: defaultPrefixes,
	}

	prefixes := []dyn.Path{dyn.MustPathFromString("variables")}
	varPath := dyn.NewPath(dyn.Key("var"))

	suggestion := m.suggest("var.my_cluster_id", normalized, prefixes, varPath)
	assert.Equal(t, "", suggestion)
}

package dresources

import (
	"reflect"
	"strings"
	"testing"

	"github.com/databricks/cli/bundle/deployplan"
	"github.com/databricks/cli/libs/structs/structpath"
	"github.com/databricks/databricks-sdk-go/service/vectorsearch"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestVectorSearchIndexAllSDKFieldsAreClassified guards against a future SDK
// bump silently adding a field that the planner classifies as Update. The
// resource has no update API and intentionally omits DoUpdate, so any
// unclassified field would surface as a deploy-time framework error
// ("resource does not support update action but plan produced update"). This
// test catches the gap at unit-test time instead.
func TestVectorSearchIndexAllSDKFieldsAreClassified(t *testing.T) {
	config := GetResourceConfig("vector_search_indexes")
	require.NotNil(t, config)

	classified := map[string]bool{}
	for _, field := range config.RecreateOnChanges {
		classified[field.Field.String()] = true
	}
	for _, field := range config.IgnoreRemoteChanges {
		classified[field.Field.String()] = true
	}

	sdkType := reflect.TypeFor[vectorsearch.CreateVectorIndexRequest]()
	for field := range sdkType.Fields() {
		jsonTag := field.Tag.Get("json")
		if jsonTag == "" || jsonTag == "-" {
			continue
		}
		jsonTag = strings.TrimSuffix(jsonTag, ",omitempty")
		assert.Truef(t, classified[jsonTag],
			"field %q is not declared in resources.yml under vector_search_indexes; "+
				"vector_search_indexes has no update API, so every SDK field must be in "+
				"recreate_on_changes or ignore_remote_changes",
			jsonTag,
		)
	}
}

func TestNormalizeColumnType(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{"int", "integer"},
		{"integer", "integer"},
		{"bigint", "long"},
		{"long", "long"},
		{"smallint", "short"},
		{"tinyint", "byte"},
		{"float", "float"},
		{"string", "string"},
		{"array<int>", "array<integer>"},
		{"array<float>", "array<float>"},
		{"array<array<bigint>>", "array<array<long>>"},
	}
	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			assert.Equal(t, tt.want, normalizeColumnType(tt.in))
		})
	}
}

func TestSchemaTypesEqual(t *testing.T) {
	tests := []struct {
		name string
		a    string
		b    string
		want bool
	}{
		{"alias equals canonical", `{"id":"int"}`, `{"id":"integer"}`, true},
		{"key order is irrelevant", `{"a":"int","b":"string"}`, `{"b":"string","a":"integer"}`, true},
		{"array alias equals canonical", `{"v":"array<int>"}`, `{"v":"array<integer>"}`, true},
		{"different type", `{"id":"int"}`, `{"id":"string"}`, false},
		{"different columns", `{"id":"int"}`, `{"id":"int","x":"string"}`, false},
		{"malformed input", `not json`, `{"id":"int"}`, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, schemaTypesEqual(tt.a, tt.b))
		})
	}
}

func TestVectorSearchIndexOverrideChangeDescSchemaJSON(t *testing.T) {
	r := &ResourceVectorSearchIndex{}
	path, err := structpath.ParsePath("direct_access_index_spec.schema_json")
	require.NoError(t, err)

	// Alias-only difference between config and the normalized remote schema:
	// the recreate is suppressed.
	change := &ChangeDesc{
		Action: deployplan.Recreate,
		Reason: "immutable",
		New:    `{"id":"integer","vector":"array<float>"}`,
		Remote: `{"id":"int","vector":"array<float>"}`,
	}
	require.NoError(t, r.OverrideChangeDesc(t.Context(), path, change, nil))
	assert.Equal(t, deployplan.Skip, change.Action)
	assert.Equal(t, deployplan.ReasonAlias, change.Reason)

	// A genuine schema change still recreates.
	change = &ChangeDesc{
		Action: deployplan.Recreate,
		Reason: "immutable",
		New:    `{"id":"string"}`,
		Remote: `{"id":"int"}`,
	}
	require.NoError(t, r.OverrideChangeDesc(t.Context(), path, change, nil))
	assert.Equal(t, deployplan.Recreate, change.Action)
}

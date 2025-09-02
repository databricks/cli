package structwalk

import (
	"testing"

	"github.com/databricks/cli/libs/structdiff/structpath"
	"github.com/databricks/databricks-sdk-go/service/jobs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func flatten(t *testing.T, value any) map[string]any {
	results := make(map[string]any)
	err := Walk(value, func(path *structpath.PathNode, value any) {
		results[path.String()] = value
	})
	require.NoError(t, err)
	return results
}

func TestValueNil(t *testing.T) {
	assert.Empty(t, flatten(t, nil))
}

func TestValueEmptyMap(t *testing.T) {
	assert.Empty(t, flatten(t, make(map[string]int)))
}

func TestValueEmptySlice(t *testing.T) {
	assert.Empty(t, flatten(t, []string{}))
}

func TestValueInt(t *testing.T) {
	assert.Equal(t, map[string]any{"": 5}, flatten(t, 5))
}

func TestValueTypesEmpty(t *testing.T) {
	expected := map[string]any{
		".ArrayString[0]":  "",
		".ArrayString[1]":  "",
		".Array[0].X":      0,
		".Array[1].X":      0,
		".BoolField":       false,
		".EmptyTagField":   "",
		".IntField":        0,
		".Nested.X":        0,
		".ValidFieldNoTag": "",
		".valid_field":     "",
	}

	assert.Equal(t, expected, flatten(t, Types{}))
	assert.Equal(t, expected, flatten(t, &Types{}))

	// Variant with ForceSendFields on some omitempty fields
	forced := Types{
		ForceSendFields: []string{"OmitStr", "OmitBool"},
	}

	forcedResults := flatten(t, forced)

	// Ensure forced fields are present with zero values
	assert.Equal(t, "", forcedResults[".omit_str"])
	assert.Equal(t, false, forcedResults[".omit_bool"])

	// Non-forced omitempty zero field should remain absent
	_, ok := forcedResults[".omit_int"]
	assert.False(t, ok, "omit_int should be absent when not forced")
}

func TestValueJobSettings(t *testing.T) {
	jobSettings := jobs.JobSettings{
		Name:              "test-job",
		MaxConcurrentRuns: 5,
		TimeoutSeconds:    3600,
		Tags:              map[string]string{"env": "test", "team": "data"},
	}

	assert.Equal(t, map[string]any{
		`.tags["env"]`:         "test",
		`.tags["team"]`:        "data",
		".name":                "test-job",
		".max_concurrent_runs": 5,
		".timeout_seconds":     3600,
	}, flatten(t, jobSettings))
}

func TestValueBundleTag(t *testing.T) {
	type Foo struct {
		A string `bundle:"readonly"`
		B string `bundle:"internal"`
		C string
		D string `bundle:"internal,readonly"`
	}

	var readonly, internal []string
	err := Walk(Foo{
		A: "a",
		B: "b",
		C: "c",
		D: "d",
	}, func(path *structpath.PathNode, value any) {
		if path.BundleTag().ReadOnly() {
			readonly = append(readonly, path.String())
		}
		if path.BundleTag().Internal() {
			internal = append(internal, path.String())
		}
	})
	require.NoError(t, err)

	assert.Equal(t, []string{".A", ".D"}, readonly)
	assert.Equal(t, []string{".B", ".D"}, internal)
}

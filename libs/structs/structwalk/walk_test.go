package structwalk

import (
	"reflect"
	"testing"

	"github.com/databricks/cli/libs/structs/structpath"
	"github.com/databricks/cli/libs/structs/structtag"
	"github.com/databricks/databricks-sdk-go/service/jobs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func flatten(t *testing.T, value any) map[string]any {
	results := make(map[string]any)
	err := Walk(value, func(path *structpath.PathNode, value any, field *reflect.StructField) {
		s := path.String()
		results[s] = value

		// Test path parsing round trip
		newPath, err := structpath.Parse(s)
		if assert.NoError(t, err, s) {
			newS := newPath.String()
			assert.Equal(t, path, newPath, "s=%q newS=%q", s, newS)
			assert.Equal(t, s, newS)
		}
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

func TestValueNonEmptyMap(t *testing.T) {
	assert.Equal(t, map[string]any{"['hello']": 5}, flatten(t, map[string]int{"hello": 5}))
}

func TestValueEmptySlice(t *testing.T) {
	assert.Empty(t, flatten(t, []string{}))
}

func TestValueInt(t *testing.T) {
	assert.Equal(t, map[string]any{"": 5}, flatten(t, 5))
}

func TestValueTypesEmpty(t *testing.T) {
	expected := map[string]any{
		"ArrayString[0]":  "",
		"ArrayString[1]":  "",
		"Array[0].X":      0,
		"Array[1].X":      0,
		"BoolField":       false,
		"EmptyTagField":   "",
		"IntField":        0,
		"Nested.X":        0,
		"ValidFieldNoTag": "",
		"valid_field":     "",
	}

	assert.Equal(t, expected, flatten(t, Types{}))
	assert.Equal(t, expected, flatten(t, &Types{}))

	// Variant with ForceSendFields on some omitempty fields
	forced := Types{
		ForceSendFields: []string{"OmitStr", "OmitBool"},
	}

	forcedResults := flatten(t, forced)

	// Ensure forced fields are present with zero values
	assert.Equal(t, "", forcedResults["omit_str"])
	assert.Equal(t, false, forcedResults["omit_bool"])

	// Non-forced omitempty zero field should remain absent
	_, ok := forcedResults["omit_int"]
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
		`tags['env']`:         "test",
		`tags['team']`:        "data",
		"name":                "test-job",
		"max_concurrent_runs": 5,
		"timeout_seconds":     3600,
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
	}, func(path *structpath.PathNode, value any, field *reflect.StructField) {
		if field == nil {
			return
		}

		bundleTag := structtag.BundleTag(field.Tag.Get("bundle"))
		if bundleTag.ReadOnly() {
			readonly = append(readonly, path.String())
		}
		if bundleTag.Internal() {
			internal = append(internal, path.String())
		}
	})
	require.NoError(t, err)

	assert.Equal(t, []string{"A", "D"}, readonly)
	assert.Equal(t, []string{"B", "D"}, internal)
}

func TestEmbeddedStruct(t *testing.T) {
	type Embedded struct {
		EmbeddedField string `json:"embedded_field"`
		EmbeddedInt   int    `json:"embedded_int"`
	}

	type Parent struct {
		Embedded
		ParentField string `json:"parent_field"`
	}

	parent := Parent{
		Embedded: Embedded{
			EmbeddedField: "embedded_value",
			EmbeddedInt:   42,
		},
		ParentField: "parent_value",
	}

	result := flatten(t, parent)

	// Embedded struct fields should be at the same level as parent fields
	assert.Equal(t, map[string]any{
		"embedded_field": "embedded_value",
		"embedded_int":   42,
		"parent_field":   "parent_value",
	}, result)
}

func TestNestedEmbeddedStructs(t *testing.T) {
	type Level1 struct {
		Field1 string `json:"field1"`
	}

	type Level2 struct {
		Level1
		Field2 string `json:"field2"`
	}

	type Level3 struct {
		Level2
		Field3 string `json:"field3"`
	}

	obj := Level3{
		Level2: Level2{
			Level1: Level1{
				Field1: "one",
			},
			Field2: "two",
		},
		Field3: "three",
	}

	assert.Equal(t, map[string]any{
		"field1": "one",
		"field2": "two",
		"field3": "three",
	}, flatten(t, obj))
}

func TestEmbeddedStructWithPointer(t *testing.T) {
	type Embedded struct {
		EmbeddedField string `json:"embedded_field"`
	}

	type Parent struct {
		*Embedded
		ParentField string `json:"parent_field"`
	}

	parent := Parent{
		Embedded: &Embedded{
			EmbeddedField: "embedded_value",
		},
		ParentField: "parent_value",
	}

	assert.Equal(t, map[string]any{
		"embedded_field": "embedded_value",
		"parent_field":   "parent_value",
	}, flatten(t, parent))
}

func TestEmbeddedStructWithJSONTagDash(t *testing.T) {
	type Embedded struct {
		SkipField    string `json:"-"`
		IncludeField string `json:"included"`
	}

	type Parent struct {
		Embedded
		ParentField string `json:"parent_field"`
	}

	parent := Parent{
		Embedded: Embedded{
			SkipField:    "should_not_appear",
			IncludeField: "should_appear",
		},
		ParentField: "parent",
	}

	assert.Equal(t, map[string]any{
		"included":     "should_appear",
		"parent_field": "parent",
	}, flatten(t, parent))
}

package structwalk

import (
	"reflect"
	"testing"

	"github.com/databricks/cli/libs/structdiff/structpath"
	"github.com/databricks/databricks-sdk-go/service/jobs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWalkTypeNilCallback(t *testing.T) {
	err := WalkType(reflect.TypeOf(""), nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "visit callback must not be nil")
}

func TestWalkTypeNilType(t *testing.T) {
	results := make(map[string]reflect.Type)
	err := WalkType(nil, func(path *structpath.PathNode, typ reflect.Type) {
		results[path.DynPath()] = typ
	})
	require.NoError(t, err)
	assert.Empty(t, results)
}

func TestWalkTypeScalarFields(t *testing.T) {
	type TestStruct struct {
		StringField string  `json:"string_field"`
		IntField    int     `json:"int_field"`
		BoolField   bool    `json:"bool_field"`
		FloatField  float64 `json:"float_field"`
	}

	results := make(map[string]reflect.Type)
	err := WalkType(reflect.TypeOf(TestStruct{}), func(path *structpath.PathNode, typ reflect.Type) {
		results[path.DynPath()] = typ
	})

	require.NoError(t, err)
	assert.Equal(t, reflect.String, results["string_field"].Kind())
	assert.Equal(t, reflect.Int, results["int_field"].Kind())
	assert.Equal(t, reflect.Bool, results["bool_field"].Kind())
	assert.Equal(t, reflect.Float64, results["float_field"].Kind())
}

func TestWalkTypePointerFields(t *testing.T) {
	type TestStruct struct {
		PointerField    *string     `json:"pointer_field"`
		PointerToSlice  *[]int      `json:"pointer_to_slice"`
		PointerToStruct *TestStruct `json:"pointer_to_struct"`
	}

	results := make(map[string]reflect.Type)
	err := WalkType(reflect.TypeOf(TestStruct{}), func(path *structpath.PathNode, typ reflect.Type) {
		results[path.DynPath()] = typ
	})

	require.NoError(t, err)

	// Pointer to scalar should be reported as pointer type
	assert.Equal(t, reflect.Pointer, results["pointer_field"].Kind())
	assert.Equal(t, reflect.String, results["pointer_field"].Elem().Kind())

	// Pointer to slice should drill down to element
	assert.Equal(t, reflect.Int, results["pointer_to_slice[0]"].Kind())

	// Pointer to struct should drill down but prevent circular reference
	assert.Equal(t, reflect.Pointer, results["pointer_to_struct.pointer_field"].Kind())
	_, found := results["pointer_to_struct.pointer_to_struct.pointer_field"]
	assert.False(t, found, "Should not find second level circular reference")
}

func TestWalkTypeCollections(t *testing.T) {
	type TestStruct struct {
		SliceField      []string         `json:"slice_field"`
		ArrayField      [2]int           `json:"array_field"`
		MapField        map[string]int   `json:"map_field"`
		SliceOfPointers []*string        `json:"slice_of_pointers"`
		MapOfSlices     map[string][]int `json:"map_of_slices"`
		IntMapField     map[int]string   `json:"int_map_field"` // Should be ignored
	}

	results := make(map[string]reflect.Type)
	err := WalkType(reflect.TypeOf(TestStruct{}), func(path *structpath.PathNode, typ reflect.Type) {
		results[path.DynPath()] = typ
	})

	require.NoError(t, err)

	// Collections should report element types
	assert.Equal(t, reflect.String, results["slice_field[0]"].Kind())
	assert.Equal(t, reflect.Int, results["array_field[0]"].Kind())
	assert.Equal(t, reflect.Int, results["map_field"].Kind())
	assert.Equal(t, reflect.Pointer, results["slice_of_pointers[0]"].Kind())
	assert.Equal(t, reflect.Int, results["map_of_slices[0]"].Kind())

	// Non-string map keys should be ignored
	_, found := results["int_map_field"]
	assert.False(t, found, "Should ignore maps with non-string keys")
}

func TestWalkTypeNestedStructs(t *testing.T) {
	type NestedStruct struct {
		Value string        `json:"value"`
		Child *NestedStruct `json:"child"`
	}

	type TestStruct struct {
		StructField NestedStruct  `json:"struct_field"`
		NodeField   *NestedStruct `json:"node_field"`
	}

	results := make(map[string]reflect.Type)
	err := WalkType(reflect.TypeOf(TestStruct{}), func(path *structpath.PathNode, typ reflect.Type) {
		results[path.DynPath()] = typ
	})

	require.NoError(t, err)

	// Nested struct fields
	assert.Equal(t, reflect.String, results["struct_field.value"].Kind())
	assert.Equal(t, reflect.String, results["struct_field.child.value"].Kind())

	// Pointer to struct with circular reference (one level only)
	assert.Equal(t, reflect.String, results["node_field.value"].Kind())
	_, found := results["node_field.child.child.value"]
	assert.False(t, found, "Should not find second level circular reference")
}

func TestWalkTypeIgnoredFields(t *testing.T) {
	type TestStruct struct {
		ValidField      string        `json:"valid_field"`
		IgnoredField    string        `json:"-"`
		DashField       string        `json:"-,omitempty"`
		NoTagField      string        // no json tag
		EmptyTagField   string        `json:""`
		unexportedField string        `json:"unexported"`
		FuncField       func() string `json:"func_field"`
		ChanField       chan string   `json:"chan_field"`
		InterfaceField  any           `json:"interface_field"`
	}

	results := make(map[string]reflect.Type)
	err := WalkType(reflect.TypeOf(TestStruct{}), func(path *structpath.PathNode, typ reflect.Type) {
		results[path.DynPath()] = typ
	})

	require.NoError(t, err)

	// Only valid field should be found
	assert.Equal(t, 1, len(results))
	assert.Equal(t, reflect.String, results["valid_field"].Kind())

	// All other fields should be ignored
	ignoredPaths := []string{
		"ignored_field", "dash_field", "NoTagField", "EmptyTagField",
		"unexportedField", "func_field", "chan_field", "interface_field",
	}
	for _, path := range ignoredPaths {
		_, found := results[path]
		assert.False(t, found, "Should not find ignored field: %s", path)
	}
}

func TestWalkTypeCircularReference(t *testing.T) {
	type SelfRef struct {
		Value string   `json:"value"`
		Self  *SelfRef `json:"self"`
	}

	results := make(map[string]reflect.Type)
	err := WalkType(reflect.TypeOf(SelfRef{}), func(path *structpath.PathNode, typ reflect.Type) {
		results[path.DynPath()] = typ
	})

	require.NoError(t, err)

	// Should find first and second level but not third level
	assert.Equal(t, reflect.String, results["value"].Kind())
	assert.Equal(t, reflect.String, results["self.value"].Kind())

	// Should NOT find third level circular reference
	_, found := results["self.self.value"]
	assert.False(t, found, "Should not find third level circular reference")
}

func TestWalkTypePointerToStruct(t *testing.T) {
	type TestStruct struct {
		StringField string `json:"string_field"`
		IntField    int    `json:"int_field"`
	}

	// Test with pointer to struct type
	results := make(map[string]reflect.Type)
	err := WalkType(reflect.TypeOf(&TestStruct{}), func(path *structpath.PathNode, typ reflect.Type) {
		results[path.DynPath()] = typ
	})

	require.NoError(t, err)
	assert.Equal(t, reflect.String, results["string_field"].Kind())
	assert.Equal(t, reflect.Int, results["int_field"].Kind())
}

func TestWalkTypeEmptyStruct(t *testing.T) {
	type EmptyStruct struct{}

	results := make(map[string]reflect.Type)
	err := WalkType(reflect.TypeOf(EmptyStruct{}), func(path *structpath.PathNode, typ reflect.Type) {
		results[path.DynPath()] = typ
	})

	require.NoError(t, err)
	assert.Empty(t, results, "Should find no paths for empty struct")
}

func TestWalkTypeUnexportedFieldsOnly(t *testing.T) {
	type UnexportedStruct struct {
		unexported1 string
		unexported2 int
	}

	results := make(map[string]reflect.Type)
	err := WalkType(reflect.TypeOf(UnexportedStruct{}), func(path *structpath.PathNode, typ reflect.Type) {
		results[path.DynPath()] = typ
	})

	require.NoError(t, err)
	assert.Empty(t, results, "Should find no paths for struct with only unexported fields")
}

func TestWalkTypeJobSettings(t *testing.T) {
	jobSettingsType := reflect.TypeOf(jobs.JobSettings{})

	results := make(map[string]reflect.Type)
	err := WalkType(jobSettingsType, func(path *structpath.PathNode, typ reflect.Type) {
		results[path.DynPath()] = typ
	})

	require.NoError(t, err)
	require.NotEmpty(t, results)

	// Verify some expected basic fields exist
	expectedFields := map[string]reflect.Kind{
		"name":                reflect.String,
		"timeout_seconds":     reflect.Int,
		"max_concurrent_runs": reflect.Int,
	}

	for path, expectedKind := range expectedFields {
		typ, found := results[path]
		assert.True(t, found, "Expected path not found: %s", path)
		if found {
			assert.Equal(t, expectedKind, typ.Kind(), "Type kind mismatch for path %s", path)
		}
	}

	// Verify circular reference behavior - we should see one level of nesting
	circularPaths := []string{
		"tasks[0].for_each_task.task.task_key",
		"tasks[0].for_each_task.task.description",
		"tasks[0].for_each_task.task.timeout_seconds",
	}

	for _, path := range circularPaths {
		_, found := results[path]
		assert.True(t, found, "Expected circular reference path not found: %s", path)
	}

	// Verify we DON'T see second level circular references
	secondLevelPaths := []string{
		"tasks[0].for_each_task.task.for_each_task.task.task_key",
		"tasks[0].for_each_task.task.for_each_task.task.description",
	}

	for _, path := range secondLevelPaths {
		_, found := results[path]
		assert.False(t, found, "Should not find second level circular reference: %s", path)
	}

	// Verify we found a reasonable number of fields (it's 533 at the time of writing)
	assert.Greater(t, len(results), 500, "Expected to find many fields in JobSettings")
	assert.Less(t, len(results), 600, "Expected to find not so many fields in JobSettings")
}

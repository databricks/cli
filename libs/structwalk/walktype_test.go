package structwalk

import (
	"reflect"
	"testing"

	"github.com/databricks/cli/libs/structdiff/structpath"
	"github.com/databricks/databricks-sdk-go/service/jobs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWalkTypeSimple(t *testing.T) {
	type SimpleStruct struct {
		Name string `json:"name"`
		Age  int    `json:"age"`
	}

	simpleType := reflect.TypeOf(SimpleStruct{})
	results := make(map[string]reflect.Type)

	err := WalkType(simpleType, func(path *structpath.PathNode, typ reflect.Type) {
		results[path.DynPath()] = typ
	})

	require.NoError(t, err)
	require.Len(t, results, 2)

	nameType, found := results["name"]
	assert.True(t, found)
	assert.Equal(t, reflect.String, nameType.Kind())

	ageType, found := results["age"]
	assert.True(t, found)
	assert.Equal(t, reflect.Int, ageType.Kind())
}

func TestWalkTypeWithPointer(t *testing.T) {
	type SimpleStruct struct {
		Name string `json:"name"`
		Age  int    `json:"age"`
	}

	// Test with pointer to struct
	simpleType := reflect.TypeOf(&SimpleStruct{})

	results := make(map[string]reflect.Type)

	err := WalkType(simpleType, func(path *structpath.PathNode, typ reflect.Type) {
		results[path.DynPath()] = typ
	})

	require.NoError(t, err)
	require.Len(t, results, 2)

	// Should get the same results as with non-pointer type
	typ, found := results["name"]
	assert.True(t, found)
	assert.Equal(t, reflect.String, typ.Kind())
}

func TestWalkTypeWithNil(t *testing.T) {
	results := make(map[string]reflect.Type)

	err := WalkType(nil, func(path *structpath.PathNode, typ reflect.Type) {
		results[path.DynPath()] = typ
	})

	require.NoError(t, err)
	assert.Empty(t, results)
}

func TestWalkTypeVisitNil(t *testing.T) {
	type SimpleStruct struct {
		Name string `json:"name"`
	}
	simpleType := reflect.TypeOf(SimpleStruct{})
	err := WalkType(simpleType, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "visit callback must not be nil")
}

func TestWalkTypeWithCustomStruct(t *testing.T) {
	type NestedStruct struct {
		NestedField string `json:"nested_field"`
	}

	type TestStruct struct {
		StringField  string         `json:"string_field"`
		IntField     int            `json:"int_field"`
		BoolField    bool           `json:"bool_field"`
		SliceField   []string       `json:"slice_field"`
		MapField     map[string]int `json:"map_field"`
		StructField  NestedStruct   `json:"struct_field"`
		PointerField *string        `json:"pointer_field"`
		IgnoredField string         `json:"-"`
		NoTagField   string         // no json tag, should be ignored
	}

	testType := reflect.TypeOf(TestStruct{})
	results := make(map[string]reflect.Type)

	err := WalkType(testType, func(path *structpath.PathNode, typ reflect.Type) {
		results[path.DynPath()] = typ
	})

	require.NoError(t, err)

	// Test expected fields
	expectedFields := map[string]reflect.Kind{
		"string_field":              reflect.String,
		"int_field":                 reflect.Int,
		"bool_field":                reflect.Bool,
		"slice_field[0]":            reflect.String,
		"map_field":                 reflect.Int,
		"struct_field.nested_field": reflect.String,
	}

	for path, expectedKind := range expectedFields {
		typ, found := results[path]
		assert.True(t, found, "Expected path not found: %s", path)
		if found {
			assert.Equal(t, expectedKind, typ.Kind(), "Type kind mismatch for path %s", path)
		}
	}

	// Test pointer field - should be pointer type
	pointerType, found := results["pointer_field"]
	assert.True(t, found, "Pointer field not found")
	if found {
		assert.Equal(t, reflect.Pointer, pointerType.Kind())
		assert.Equal(t, reflect.String, pointerType.Elem().Kind())
	}

	// Test ignored fields
	_, found = results["ignored_field"]
	assert.False(t, found, "Ignored field should not be found")

	_, found = results["NoTagField"]
	assert.False(t, found, "Field without json tag should not be found")
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
	// The for_each_task contains a task field that references back to the Task type
	circularPaths := []string{
		"tasks[0].for_each_task.task.task_key",        // One level into circular reference
		"tasks[0].for_each_task.task.description",     // Another field in the circular reference
		"tasks[0].for_each_task.task.timeout_seconds", // Yet another field
	}

	for _, path := range circularPaths {
		_, found := results[path]
		assert.True(t, found, "Expected circular reference path not found: %s", path)
	}

	// Verify we DON'T see second level circular references
	// These would be paths like tasks[0].for_each_task.task.for_each_task.task.*
	secondLevelPaths := []string{
		"tasks[0].for_each_task.task.for_each_task.task.task_key",
		"tasks[0].for_each_task.task.for_each_task.task.description",
	}

	for _, path := range secondLevelPaths {
		_, found := results[path]
		assert.False(t, found, "Should not find second level circular reference: %s", path)
	}

	// Verify we found a reasonable number of fields (should be hundreds)
	assert.Greater(t, len(results), 100, "Expected to find many fields in JobSettings")

	// Log some example circular reference paths for verification
	t.Logf("Total fields found: %d", len(results))
	for _, path := range circularPaths {
		if _, found := results[path]; found {
			t.Logf("Found expected circular reference path: %s", path)
		}
	}
}

func TestWalkTypeCircularReference(t *testing.T) {
	// Define a struct that has a circular reference to itself
	type Node struct {
		Value string `json:"value"`
		Child *Node  `json:"child"`
	}

	nodeType := reflect.TypeOf(Node{})
	results := make(map[string]reflect.Type)
	paths := []string{}

	err := WalkType(nodeType, func(path *structpath.PathNode, typ reflect.Type) {
		pathStr := path.DynPath()
		results[pathStr] = typ
		paths = append(paths, pathStr)
	})

	require.NoError(t, err)
	require.NotEmpty(t, results)

	// We should see:
	// - "value" (string field)
	// - "child.value" (string field in the child node - one level of circular reference)
	// But NOT "child.child.value" (would be second level, should be stopped)

	expectedPaths := []string{"value", "child.value"}
	for _, expectedPath := range expectedPaths {
		_, found := results[expectedPath]
		assert.True(t, found, "Expected path not found: %s", expectedPath)
	}

	// Should NOT find deeper circular paths
	_, found := results["child.child.value"]
	assert.False(t, found, "Should not find second level circular reference: child.child.value")

	t.Logf("Found paths: %v", paths)
}

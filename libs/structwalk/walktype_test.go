package structwalk

import (
	"reflect"
	"testing"

	"github.com/databricks/cli/bundle/config"
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

	assert.Equal(t, reflect.Pointer, results["pointer_field"].Kind())
	assert.Equal(t, reflect.String, results["pointer_field"].Elem().Kind())
	assert.Equal(t, reflect.Int, results["pointer_to_slice[0]"].Kind())
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

	assert.Equal(t, reflect.String, results["slice_field[0]"].Kind())
	assert.Equal(t, reflect.Int, results["array_field[0]"].Kind())
	assert.Equal(t, reflect.Int, results["map_field"].Kind())
	assert.Equal(t, reflect.Pointer, results["slice_of_pointers[0]"].Kind())
	assert.Equal(t, reflect.Int, results["map_of_slices[0]"].Kind())

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

	assert.Equal(t, reflect.String, results["struct_field.value"].Kind())
	assert.Equal(t, reflect.String, results["struct_field.child.value"].Kind())
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
		unexportedField string        `json:"unexported"` //nolint
		FuncField       func() string `json:"func_field"`
		ChanField       chan string   `json:"chan_field"`
		InterfaceField  any           `json:"interface_field"`
	}

	results := make(map[string]reflect.Type)
	err := WalkType(reflect.TypeOf(TestStruct{}), func(path *structpath.PathNode, typ reflect.Type) {
		results[path.DynPath()] = typ
	})

	require.NoError(t, err)

	assert.Equal(t, 1, len(results))
	assert.Equal(t, reflect.String, results["valid_field"].Kind())

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

	assert.Equal(t, reflect.String, results["value"].Kind())
	assert.Equal(t, reflect.String, results["self.value"].Kind())

	_, found := results["self.self.value"]
	assert.False(t, found, "Should not find third level circular reference")
}

func TestWalkTypeEdgeCases(t *testing.T) {
	// Test empty struct
	type EmptyStruct struct{}
	results := make(map[string]reflect.Type)
	err := WalkType(reflect.TypeOf(EmptyStruct{}), func(path *structpath.PathNode, typ reflect.Type) {
		results[path.DynPath()] = typ
	})
	require.NoError(t, err)
	assert.Empty(t, results, "Should find no paths for empty struct")

	// Test struct with only unexported fields
	type UnexportedStruct struct {
		unexported1 string //nolint
		unexported2 int    //nolint
	}
	results = make(map[string]reflect.Type)
	err = WalkType(reflect.TypeOf(UnexportedStruct{}), func(path *structpath.PathNode, typ reflect.Type) {
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

func TestWalkTypeRootEmbeddedJobSettings(t *testing.T) {
	rootType := reflect.TypeOf(config.Root{})

	results := make(map[string]reflect.Type)
	err := WalkType(rootType, func(path *structpath.PathNode, typ reflect.Type) {
		results[path.DynPath()] = typ
	})

	require.NoError(t, err)
	require.NotEmpty(t, results)

	// Verify JobSettings fields are accessible through resources.jobs
	expectedJobSettingsFields := map[string]reflect.Kind{
		"resources.jobs.name":                reflect.String,
		"resources.jobs.timeout_seconds":     reflect.Int,
		"resources.jobs.max_concurrent_runs": reflect.Int,
		"resources.jobs.format":              reflect.String,
		"resources.jobs.description":         reflect.String,
	}

	for path, expectedKind := range expectedJobSettingsFields {
		typ, found := results[path]
		assert.True(t, found, "Expected JobSettings field not found: %s", path)
		if found {
			assert.Equal(t, expectedKind, typ.Kind(), "Type kind mismatch for JobSettings field %s", path)
		}
	}

	// Verify nested task fields are accessible
	expectedTaskFields := map[string]reflect.Kind{
		"resources.jobs.tasks[0].task_key":                                       reflect.String,
		"resources.jobs.tasks[0].description":                                    reflect.String,
		"resources.jobs.tasks[0].timeout_seconds":                                reflect.Int,
		"resources.jobs.tasks[0].max_retries":                                    reflect.Int,
		"resources.jobs.tasks[0].notebook_task.notebook_path":                    reflect.String,
		"resources.jobs.tasks[0].spark_jar_task.main_class_name":                 reflect.String,
		"resources.jobs.tasks[0].spark_python_task.python_file":                  reflect.String,
		"resources.jobs.tasks[0].sql_task.query.query_id":                        reflect.String,
		"resources.jobs.tasks[0].dbt_task.commands[0]":                           reflect.String,
		"resources.jobs.tasks[0].pipeline_task.pipeline_id":                      reflect.String,
		"resources.jobs.tasks[0].python_wheel_task.package_name":                 reflect.String,
		"resources.jobs.tasks[0].for_each_task.inputs":                           reflect.String,
		"resources.jobs.tasks[0].for_each_task.task.task_key":                    reflect.String,
		"resources.jobs.tasks[0].for_each_task.task.notebook_task.notebook_path": reflect.String,
		"resources.jobs.tasks[0].new_cluster.node_type_id":                       reflect.String,
		"resources.jobs.tasks[0].new_cluster.num_workers":                        reflect.Int,
		"resources.jobs.tasks[0].new_cluster.spark_version":                      reflect.String,
	}

	for path, expectedKind := range expectedTaskFields {
		typ, found := results[path]
		assert.True(t, found, "Expected nested task field not found: %s", path)
		if found {
			assert.Equal(t, expectedKind, typ.Kind(), "Type kind mismatch for nested task field %s", path)
		}
	}

	// Verify job cluster fields are accessible
	expectedJobClusterFields := map[string]reflect.Kind{
		"resources.jobs.job_clusters[0].job_cluster_key":           reflect.String,
		"resources.jobs.job_clusters[0].new_cluster.node_type_id":  reflect.String,
		"resources.jobs.job_clusters[0].new_cluster.num_workers":   reflect.Int,
		"resources.jobs.job_clusters[0].new_cluster.spark_version": reflect.String,
	}

	for path, expectedKind := range expectedJobClusterFields {
		typ, found := results[path]
		assert.True(t, found, "Expected job cluster field not found: %s", path)
		if found {
			assert.Equal(t, expectedKind, typ.Kind(), "Type kind mismatch for job cluster field %s", path)
		}
	}

	// Verify Root-specific fields are also present
	expectedRootFields := map[string]reflect.Kind{
		"bundle.name":                reflect.String,
		"bundle.target":              reflect.String,
		"workspace.host":             reflect.String,
		"workspace.profile":          reflect.String,
		"variables.lookup.dashboard": reflect.String,
	}

	for path, expectedKind := range expectedRootFields {
		typ, found := results[path]
		assert.True(t, found, "Expected Root field not found: %s", path)
		if found {
			assert.Equal(t, expectedKind, typ.Kind(), "Type kind mismatch for Root field %s", path)
		}
	}

	t.Logf("Total fields found in Root: %d", len(results))
}

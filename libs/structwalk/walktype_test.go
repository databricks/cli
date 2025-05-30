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

func getFields(t *testing.T, typ reflect.Type) map[string]any {
	results := make(map[string]any)
	err := WalkType(typ, func(path *structpath.PathNode, typ reflect.Type) {
		results[path.String()] = reflect.Zero(typ).Interface()
	})
	require.NoError(t, err)
	return results
}

func TestTypeNilCallback(t *testing.T) {
	err := WalkType(reflect.TypeOf(""), nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "visit callback must not be nil")
}

func TestTypeNil(t *testing.T) {
	assert.Equal(t, map[string]any{}, getFields(t, reflect.TypeOf(nil)))
}

func TestTypeScalar(t *testing.T) {
	assert.Equal(t, map[string]any{"": 0}, getFields(t, reflect.TypeOf(5)))
}

func TestTypes(t *testing.T) {
	assert.Equal(t, map[string]any{
		".ArrayString[0]":     "",
		".Array[0].X":         0,
		".BoolField":          false,
		".EmptyTagField":      "",
		".EmptyTagFieldPtr":   "",
		".IntField":           0,
		".Map[\"\"].X":        0, /// XXX bad
		".MapPtr[\"\"].X":     0,
		".Nested.X":           0,
		".NestedPtr.X":        0,
		".SliceString[0]":     "",
		".Slice[0].X":         0,
		".ValidFieldNoTag":    "",
		".ValidFieldPtrNoTag": "",
		".omit_bool":          false,
		".omit_int":           0,
		".omit_str":           "",
		".valid_field":        "",
		".valid_field_ptr":    "",
	}, getFields(t, reflect.TypeOf(Types{})))
}

func TestTypeSelf(t *testing.T) {
	assert.Equal(t, map[string]any{
		".valid_field":                   "",
		".SelfArrayPtr[0].valid_field":   "",
		".SelfIndirect.X.valid_field":    "",
		".SelfIndirectPtr.X.valid_field": "",
		".SelfMapPtr[\"\"].valid_field":  "",
		".SelfMap[\"\"].valid_field":     "",
		".SelfReference.valid_field":     "",
		".SelfSlicePtr[0].valid_field":   "",
		".SelfSlice[0].valid_field":      "",
	}, getFields(t, reflect.TypeOf(Self{})))
}

func testStruct(t *testing.T, typ reflect.Type, minLen, maxLen int, present map[string]any, notPresent []string) {
	results := getFields(t, typ)

	assert.Greater(t, len(results), minLen, "Expected to find many fields in %s", typ)
	assert.Less(t, len(results), maxLen, "Expected to find not so many fields in %s", typ)

	for path, expectedValue := range present {
		value, found := results[path]
		assert.True(t, found, "Expected path not found in %s: %s", typ, path)
		assert.Equal(t, expectedValue, value, "%s %s", typ, path)
	}

	for _, path := range notPresent {
		_, found := results[path]
		assert.False(t, found, "Should not find %s in %s", path, typ)
	}
}

func TestTypeJobSettings(t *testing.T) {
	testStruct(t,
		reflect.TypeOf(jobs.JobSettings{}),
		// Verify we found a reasonable number of fields (it's 533 at the time of writing)
		500, 600,
		map[string]any{
			".name":                "",
			".timeout_seconds":     0,
			".max_concurrent_runs": 0,

			// Verify circular reference behavior - we should see one level of nesting
			".tasks[0].for_each_task.task.task_key":        "",
			".tasks[0].for_each_task.task.description":     "",
			".tasks[0].for_each_task.task.timeout_seconds": 0,
		},

		// Verify we DON'T see second level circular references
		[]string{
			".tasks[0].for_each_task.task.for_each_task.task.task_key",
			".tasks[0].for_each_task.task.for_each_task.task.description",
		},
	)
}

func TestTypeRoot(t *testing.T) {
	testStruct(t,
		reflect.TypeOf(config.Root{}),
		3400, 3500, // 3487 at this time
		map[string]any{
			".bundle.target":                  "",
			`.variables[""].lookup.dashboard`: "",

			`.resources.jobs[""].name`:                "",
			`.resources.jobs[""].timeout_seconds`:     0,
			`.resources.jobs[""].max_concurrent_runs`: 0,
			`.resources.jobs[""].format`:              jobs.Format(""),
			`.resources.jobs[""].description`:         "",

			// Verify nested task fields are accessible
			`.resources.jobs[""].tasks[0].task_key`:                                       "",
			`.resources.jobs[""].tasks[0].notebook_task.notebook_path`:                    "",
			`.resources.jobs[""].tasks[0].spark_jar_task.main_class_name`:                 "",
			`.resources.jobs[""].tasks[0].for_each_task.inputs`:                           "",
			`.resources.jobs[""].tasks[0].for_each_task.task.task_key`:                    "",
			`.resources.jobs[""].tasks[0].for_each_task.task.notebook_task.notebook_path`: "",
			`.resources.jobs[""].tasks[0].new_cluster.node_type_id`:                       "",
			`.resources.jobs[""].tasks[0].new_cluster.num_workers`:                        0,

			// Verify job cluster fields are accessible
			`.resources.jobs[""].job_clusters[0].job_cluster_key`:         "",
			`.resources.jobs[""].job_clusters[0].new_cluster.num_workers`: 0,
		},
		nil,
	)
}

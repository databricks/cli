package structwalk

import (
	"reflect"
	"strings"
	"testing"

	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/libs/structs/structpath"
	"github.com/databricks/databricks-sdk-go/service/jobs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func getScalarFields(t *testing.T, typ reflect.Type) map[string]any {
	results := make(map[string]any)
	err := WalkType(typ, func(path *structpath.PatternNode, typ reflect.Type, field *reflect.StructField) (continueWalk bool) {
		for typ.Kind() == reflect.Pointer {
			typ = typ.Elem()
		}

		s := path.String()

		if isScalar(typ.Kind()) {
			results[s] = reflect.Zero(typ).Interface()
		}

		// Test structpath round trip as well
		pathNew, err := structpath.ParsePattern(s)
		if assert.NoError(t, err, "ParsePattern(path.String()) failed for %q: %s", s, err) {
			newS := pathNew.String()
			assert.Equal(t, path, pathNew, "Parse(path.String()) returned different path;\npath=%#v %q\npathNew=%#v %q", path, s, pathNew, newS)
			assert.Equal(t, s, newS, "Parse(path.String()).String() is different from path.String()\npath.String()=%q\npathNew.String()=%q", path, pathNew)
		}

		return true
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
	assert.Equal(t, map[string]any{}, getScalarFields(t, reflect.TypeOf(nil)))
}

func TestTypeScalar(t *testing.T) {
	assert.Equal(t, map[string]any{"": 0}, getScalarFields(t, reflect.TypeOf(5)))
}

func TestTypes(t *testing.T) {
	assert.Equal(t, map[string]any{
		"ArrayString[*]":     "",
		"Array[*].X":         0,
		"BoolField":          false,
		"EmptyTagField":      "",
		"EmptyTagFieldPtr":   "",
		"IntField":           0,
		`Map.*.X`:            0,
		`MapPtr.*.X`:         0,
		"Nested.X":           0,
		"NestedPtr.X":        0,
		"SliceString[*]":     "",
		"Slice[*].X":         0,
		"ValidFieldNoTag":    "",
		"ValidFieldPtrNoTag": "",
		"omit_bool":          false,
		"omit_int":           0,
		"omit_str":           "",
		"valid_field":        "",
		"valid_field_ptr":    "",
	}, getScalarFields(t, reflect.TypeOf(Types{})))
}

func TestTypeSelf(t *testing.T) {
	assert.Equal(t, map[string]any{
		"valid_field":                   "",
		"SelfArrayPtr[*].valid_field":   "",
		"SelfIndirect.X.valid_field":    "",
		"SelfIndirectPtr.X.valid_field": "",
		`SelfMapPtr.*.valid_field`:      "",
		`SelfMap.*.valid_field`:         "",
		"SelfReference.valid_field":     "",
		"SelfSlicePtr[*].valid_field":   "",
		"SelfSlice[*].valid_field":      "",
	}, getScalarFields(t, reflect.TypeOf(Self{})))
}

func testStruct(t *testing.T, typ reflect.Type, minLen, maxLen int, present map[string]any, notPresent []string) {
	results := getScalarFields(t, typ)

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
			"name":                "",
			"timeout_seconds":     0,
			"max_concurrent_runs": 0,

			// Verify circular reference behavior - we should see one level of nesting
			"tasks[*].for_each_task.task.task_key":        "",
			"tasks[*].for_each_task.task.description":     "",
			"tasks[*].for_each_task.task.timeout_seconds": 0,
		},

		// Verify we DON'T see second level circular references
		[]string{
			"tasks[*].for_each_task.task.for_each_task.task.task_key",
			"tasks[*].for_each_task.task.for_each_task.task.description",
		},
	)
}

func TestTypeRoot(t *testing.T) {
	testStruct(t,
		reflect.TypeOf(config.Root{}),
		4600, 5000, // 4814 after adding external locations support
		map[string]any{
			"bundle.target":                "",
			`variables.*.lookup.dashboard`: "",

			`resources.jobs.*.name`:                "",
			`resources.jobs.*.timeout_seconds`:     0,
			`resources.jobs.*.max_concurrent_runs`: 0,
			`resources.jobs.*.format`:              jobs.Format(""),
			`resources.jobs.*.description`:         "",

			// Verify nested task fields are accessible
			`resources.jobs.*.tasks[*].task_key`:                                       "",
			`resources.jobs.*.tasks[*].notebook_task.notebook_path`:                    "",
			`resources.jobs.*.tasks[*].spark_jar_task.main_class_name`:                 "",
			`resources.jobs.*.tasks[*].for_each_task.inputs`:                           "",
			`resources.jobs.*.tasks[*].for_each_task.task.task_key`:                    "",
			`resources.jobs.*.tasks[*].for_each_task.task.notebook_task.notebook_path`: "",
			`resources.jobs.*.tasks[*].new_cluster.node_type_id`:                       "",
			`resources.jobs.*.tasks[*].new_cluster.num_workers`:                        0,

			// Verify job cluster fields are accessible
			`resources.jobs.*.job_clusters[*].job_cluster_key`:         "",
			`resources.jobs.*.job_clusters[*].new_cluster.num_workers`: 0,
		},
		nil,
	)
}

func getReadonlyFields(t *testing.T, rootType reflect.Type) []string {
	var results []string
	err := WalkType(rootType, func(path *structpath.PatternNode, typ reflect.Type, field *reflect.StructField) (continueWalk bool) {
		if path == nil || field == nil {
			return true
		}
		bundleTag := field.Tag.Get("bundle")
		if strings.Contains(bundleTag, "readonly") {
			results = append(results, path.String())
		}
		return true
	})
	require.NoError(t, err)
	return results
}

func TestTypeReadonlyFields(t *testing.T) {
	readonlyFields := getReadonlyFields(t, reflect.TypeOf(config.Root{}))

	expected := []string{
		"bundle.mode",
		"bundle.target",
		"resources.jobs.*.id",
		"resources.pipelines.*.id",
		"workspace.current_user.short_name",
	}

	for _, v := range expected {
		assert.Contains(t, readonlyFields, v)
	}
}

func TestTypeBundleTag(t *testing.T) {
	type Foo struct {
		A string `bundle:"readonly"`
		B string `bundle:"internal"`
		C string
		D string `bundle:"internal,readonly"`
	}

	var readonly, internal []string
	err := WalkType(reflect.TypeOf(Foo{}), func(path *structpath.PatternNode, typ reflect.Type, field *reflect.StructField) (continueWalk bool) {
		if path == nil || field == nil {
			return true
		}
		bundleTag := field.Tag.Get("bundle")
		if strings.Contains(bundleTag, "readonly") {
			readonly = append(readonly, path.String())
		}
		if strings.Contains(bundleTag, "internal") {
			internal = append(internal, path.String())
		}
		return true
	})
	require.NoError(t, err)

	assert.Equal(t, []string{"A", "D"}, readonly)
	assert.Equal(t, []string{"B", "D"}, internal)
}

func TestWalkTypeVisited(t *testing.T) {
	type Inner struct {
		A int
		B ***int
	}

	type Outer struct {
		Inner      Inner
		MapInner   map[string]*Inner
		SliceInner []Inner

		C string
		D bool
	}

	var visited []string
	err := WalkType(reflect.TypeOf(Outer{}), func(path *structpath.PatternNode, typ reflect.Type, field *reflect.StructField) (continueWalk bool) {
		if path == nil {
			return true
		}
		visited = append(visited, path.String())
		return true
	})
	require.NoError(t, err)

	assert.Equal(t, []string{
		"Inner",
		"Inner.A",
		"Inner.B",
		"MapInner",
		"MapInner.*",
		"MapInner.*.A",
		"MapInner.*.B",
		"SliceInner",
		"SliceInner[*]",
		"SliceInner[*].A",
		"SliceInner[*].B",
		"C",
		"D",
	}, visited)
}

func TestWalkSkip(t *testing.T) {
	type Outer struct {
		A int
		B int

		Inner struct {
			C int
		}

		D int
	}

	var seen []string
	err := WalkType(reflect.TypeOf(Outer{}), func(path *structpath.PatternNode, typ reflect.Type, field *reflect.StructField) (continueWalk bool) {
		if path == nil {
			return true
		}
		seen = append(seen, path.String())
		return path.String() != "Inner"
	})
	require.NoError(t, err)
	assert.Equal(t, []string{"A", "B", "Inner", "D"}, seen)
}

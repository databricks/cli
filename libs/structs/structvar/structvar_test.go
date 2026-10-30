package structvar_test

import (
	"reflect"
	"strings"
	"testing"

	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/structs/structpath"
	"github.com/databricks/cli/libs/structs/structvar"
	"github.com/databricks/databricks-sdk-go/service/jobs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type TestObj struct {
	Name string            `json:"name"`
	Age  int               `json:"age"`
	Tags map[string]string `json:"tags"`
}

// newTestStructVar creates a fresh StructVar instance for testing
func newTestStructVar(t *testing.T) *structvar.StructVar {
	return structvar.NewStructVar(
		&TestObj{
			Name: "OldName",
			Age:  25,
			Tags: map[string]string{
				"env": "old_env",
			},
		},
		map[string]string{
			"name":            "${var.name}",
			"age":             "${var.age}",
			"tags['version']": "${var.version}",
		},
	)
}

func TestResolveRef(t *testing.T) {
	tests := []struct {
		name         string
		setup        func() *structvar.StructVar // custom setup for special cases
		reference    string
		value        any
		expectedObj  *TestObj
		expectedRefs map[string]string
		errorMsg     string // if set, test expects an error containing this message
	}{
		{
			name:      "resolve simple field reference",
			reference: "${var.name}",
			value:     "NewName",
			expectedObj: &TestObj{
				Name: "NewName",
				Age:  25,
				Tags: map[string]string{
					"env": "old_env",
				},
			},
			expectedRefs: map[string]string{
				"age":             "${var.age}",
				"tags['version']": "${var.version}",
			},
		},
		{
			name:      "resolve age field reference",
			reference: "${var.age}",
			value:     99,
			expectedObj: &TestObj{
				Name: "OldName",
				Age:  99,
				Tags: map[string]string{
					"env": "old_env",
				},
			},
			expectedRefs: map[string]string{
				"name":            "${var.name}",
				"tags['version']": "${var.version}",
			},
		},
		{
			name:      "resolve map field reference",
			reference: "${var.version}",
			value:     "new_version",
			expectedObj: &TestObj{
				Name: "OldName",
				Age:  25,
				Tags: map[string]string{
					"env":     "old_env",
					"version": "new_version",
				},
			},
			expectedRefs: map[string]string{
				"name": "${var.name}",
				"age":  "${var.age}",
			},
		},
		{
			name:      "reference not found returns error",
			reference: "${var.nonexistent}",
			value:     "NewName",
			errorMsg:  "reference not found",
		},
		{
			name: "error on invalid path",
			setup: func() *structvar.StructVar {
				return structvar.NewStructVar(
					&TestObj{},
					map[string]string{
						"invalid[path": "${var.test}",
					},
				)
			},
			reference: "${var.test}",
			value:     "value",
			errorMsg:  "invalid path",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create instance for this test
			var sv *structvar.StructVar
			if tt.setup != nil {
				sv = tt.setup()
			} else {
				sv = newTestStructVar(t)
			}

			err := sv.ResolveRef(tt.reference, tt.value)

			if tt.errorMsg != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.expectedObj, sv.Value)
			assert.Equal(t, tt.expectedRefs, sv.Refs)
		})
	}
}

func TestResolveRefMultiReference(t *testing.T) {
	sv := structvar.NewStructVar(
		&TestObj{
			Name: "OldName",
		},
		map[string]string{
			"name": "${var.prefix} ${var.suffix}",
		},
	)

	// Resolve one reference
	err := sv.ResolveRef("${var.prefix}", "Hello")
	require.NoError(t, err)

	// The value should be partially resolved
	assert.Equal(t, "Hello ${var.suffix}", sv.Value.(*TestObj).Name)
	assert.Equal(t, map[string]string{
		"name": "Hello ${var.suffix}", // partially resolved
	}, sv.Refs)

	// Resolve the remaining reference
	err = sv.ResolveRef("${var.suffix}", "World")
	require.NoError(t, err)

	// Now it should be fully resolved
	assert.Equal(t, "Hello World", sv.Value.(*TestObj).Name)
	assert.Equal(t, map[string]string{}, sv.Refs) // fully resolved, reference removed
}

func TestResolveRefEscaped(t *testing.T) {
	tests := []struct {
		name     string
		template string
		want     string
	}{
		{
			name:     "escaped placeholder beside a real reference",
			template: "$${runtime} ${var.name}",
			want:     "${runtime} Resolved",
		},
		{
			name:     "same name escaped first must not be substituted",
			template: "$${var.name} ${var.name}",
			want:     "${var.name} Resolved",
		},
		{
			name:     "same name real first",
			template: "${var.name} $${var.name}",
			want:     "Resolved ${var.name}",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sv := structvar.NewStructVar(
				&TestObj{Name: tt.template},
				map[string]string{"name": tt.template},
			)

			require.NoError(t, sv.ResolveRef("${var.name}", "Resolved"))

			// The struct holds the unescaped value that is sent to the API, and the
			// entry is dropped because the only reference left is a literal.
			assert.Equal(t, tt.want, sv.Value.(*TestObj).Name)
			assert.Empty(t, sv.Refs)
		})
	}
}

func TestResolveRefJobSettings(t *testing.T) {
	// Create a realistic JobSettings based on the provided YAML config
	jobSettings := jobs.JobSettings{
		Name: "job foo",
		Tasks: []jobs.Task{
			{
				TaskKey: "job_task",
				RunJobTask: &jobs.RunJobTask{
					JobId: 0, // This will be resolved from the reference
				},
			},
		},
	}

	sv := structvar.NewStructVar(
		&jobSettings,
		map[string]string{
			"tasks[0].run_job_task.job_id": "${resources.jobs.bar.id}",
		},
	)

	// Resolve the reference with a realistic job ID value (as string that gets converted to int64)
	err := sv.ResolveRef("${resources.jobs.bar.id}", "123")
	require.NoError(t, err)

	// Verify the job ID was set correctly
	updatedSettings := sv.Value.(*jobs.JobSettings)
	assert.Equal(t, "job foo", updatedSettings.Name)
	assert.Len(t, updatedSettings.Tasks, 1)
	assert.Equal(t, "job_task", updatedSettings.Tasks[0].TaskKey)
	assert.NotNil(t, updatedSettings.Tasks[0].RunJobTask)
	assert.Equal(t, int64(123), updatedSettings.Tasks[0].RunJobTask.JobId)

	// Verify the reference was removed after resolution
	assert.Empty(t, sv.Refs)
}

func TestToJSONAndBack(t *testing.T) {
	original := structvar.NewStructVar(
		&TestObj{
			Name: "TestName",
			Age:  30,
			Tags: map[string]string{
				"env": "prod",
			},
		},
		map[string]string{
			"name": "${var.name}",
		},
	)

	// Convert to JSON
	svJSON, err := original.ToJSON()
	require.NoError(t, err)
	assert.NotEmpty(t, svJSON.Value)
	assert.Equal(t, original.Refs, svJSON.Refs)

	// Convert back to StructVar
	restored, err := svJSON.ToStructVar(reflect.TypeFor[*TestObj]())
	require.NoError(t, err)

	// Verify the restored value matches
	assert.Equal(t, original.Value, restored.Value)
	assert.Equal(t, original.Refs, restored.Refs)
}

func TestToStructVarRequiresPointerType(t *testing.T) {
	svJSON := &structvar.StructVarJSON{
		Value: []byte(`{"name":"test"}`),
	}

	// Should fail with non-pointer type
	_, err := svJSON.ToStructVar(reflect.TypeFor[TestObj]())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "expecting pointer")

	// Should succeed with pointer type
	sv, err := svJSON.ToStructVar(reflect.TypeFor[*TestObj]())
	require.NoError(t, err)
	assert.Equal(t, "test", sv.Value.(*TestObj).Name)
}

func TestMoveMetadata(t *testing.T) {
	type obj struct {
		Old map[string]TestObj `json:"old"`
		New map[string]TestObj `json:"new"`
	}
	sv := structvar.NewStructVar(&obj{}, map[string]string{
		"old['a'].age": "${var.age}",
		"older.age":    "${var.other}",
		"new['b'].age": "${var.replaced}",
	})
	loc := diag.Location{File: "a.yml", Line: 1, Column: 1}
	sv.Locations = sv.Locations.With(structpath.NewStringKey(nil, "old"), (*structvar.Locations)(nil).WithLocations([]diag.Location{loc}))

	sv.MoveMetadata(structpath.NewStringKey(nil, "old"), structpath.NewStringKey(nil, "new"))

	assert.Equal(t, map[string]string{
		"new['a'].age": "${var.age}",
		"older.age":    "${var.other}",
	}, sv.Refs)
	assert.Equal(t, []diag.Location{loc}, sv.Locations.Exact(structpath.NewStringKey(nil, "new")))
	assert.Nil(t, sv.Locations.Sub(structpath.NewStringKey(nil, "old")))
}

func TestCopyMetadata(t *testing.T) {
	type obj struct {
		Items map[string]TestObj `json:"items"`
	}
	dst := structvar.NewStructVar(&obj{}, map[string]string{
		"items.a.age": "${var.old}",
		"items.b.age": "${var.kept}",
	})
	src := structvar.NewStructVar(&obj{}, map[string]string{
		"items.a.name": "${var.new}",
		"items.b.age":  "${var.ignored}",
	})
	loc := diag.Location{File: "a.yml", Line: 2, Column: 3}
	a := structpath.NewPath(nil, "items", "a")
	src.Locations = src.Locations.With(a, (*structvar.Locations)(nil).WithLocations([]diag.Location{loc}))

	dst.CopyMetadata(src, a)

	assert.Equal(t, map[string]string{
		"items.a.name": "${var.new}",
		"items.b.age":  "${var.kept}",
	}, dst.Refs)
	assert.Equal(t, []diag.Location{loc}, dst.Locations.Exact(a))
}

func TestDeleteSliceElement(t *testing.T) {
	type obj struct {
		Items []string `json:"items"`
	}
	sv := &structvar.StructVar{Value: &obj{Items: []string{"a", "b", "c"}}}
	items := structpath.NewStringKey(nil, "items")
	for i, name := range []string{"a", "b", "c"} {
		loc := diag.Location{File: name, Line: i + 1}
		sv.Locations = sv.Locations.With(structpath.NewIndex(items, i), (*structvar.Locations)(nil).WithLocations([]diag.Location{loc}))
	}
	sv.Refs = map[string]string{"items[2]": "${var.c}"}

	require.NoError(t, sv.Delete(structpath.NewIndex(items, 0)))

	assert.Equal(t, []string{"b", "c"}, sv.Value.(*obj).Items)
	assert.Equal(t, []diag.Location{{File: "b", Line: 2}}, sv.Locations.Exact(structpath.NewIndex(items, 0)))
	assert.Equal(t, []diag.Location{{File: "c", Line: 3}}, sv.Locations.Exact(structpath.NewIndex(items, 1)))
	assert.Nil(t, sv.Locations.Exact(structpath.NewIndex(items, 2)))
	assert.Equal(t, map[string]string{"items[1]": "${var.c}"}, sv.Refs)

	require.NoError(t, sv.Delete(structpath.NewIndex(items, 1)))
	assert.Equal(t, []string{"b"}, sv.Value.(*obj).Items)
	assert.Nil(t, sv.Locations.Exact(structpath.NewIndex(items, 1)))
	assert.Empty(t, sv.Refs)
}

func TestMergeNullKeepsFieldSet(t *testing.T) {
	type inner struct {
		Enabled bool `json:"enabled"`
	}
	type obj struct {
		Name            string   `json:"name,omitempty"`
		Queue           *inner   `json:"queue,omitempty"`
		ForceSendFields []string `json:"-"`
	}

	var dst obj
	dstVar, _, err := structvar.DecodeYAML("dst.yml", strings.NewReader("name: x\n"), &dst)
	require.NoError(t, err)

	var src obj
	srcVar, _, err := structvar.DecodeYAML("src.yml", strings.NewReader("queue: null\n"), &src)
	require.NoError(t, err)

	require.NoError(t, dstVar.Merge(nil, srcVar.View()))

	// An explicit null added by the merge stays set, as it does when it is decoded.
	assert.Nil(t, dst.Queue)
	assert.Equal(t, []string{"Queue"}, dst.ForceSendFields)
}

func TestMergeNullIntoValueKeepsIt(t *testing.T) {
	type inner struct {
		Enabled bool `json:"enabled"`
	}
	type obj struct {
		Queue           *inner   `json:"queue,omitempty"`
		ForceSendFields []string `json:"-"`
	}

	var dst obj
	dstVar, _, err := structvar.DecodeYAML("dst.yml", strings.NewReader("queue:\n  enabled: true\n"), &dst)
	require.NoError(t, err)

	var src obj
	srcVar, _, err := structvar.DecodeYAML("src.yml", strings.NewReader("queue: null\n"), &src)
	require.NoError(t, err)

	require.NoError(t, dstVar.Merge(nil, srcVar.View()))

	// A null does not clear an existing value.
	require.NotNil(t, dst.Queue)
	assert.True(t, dst.Queue.Enabled)
}

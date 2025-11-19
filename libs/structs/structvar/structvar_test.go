package structvar_test

import (
	"testing"

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
func newTestStructVar() *structvar.StructVar {
	return &structvar.StructVar{
		Value: &TestObj{
			Name: "OldName",
			Age:  25,
			Tags: map[string]string{
				"env": "old_env",
			},
		},
		Refs: map[string]string{
			"name":            "${var.name}",
			"age":             "${var.age}",
			"tags['version']": "${var.version}",
		},
	}
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
				return &structvar.StructVar{
					Value: &TestObj{},
					Refs: map[string]string{
						"invalid[path": "${var.test}",
					},
				}
			},
			reference: "${var.test}",
			value:     "value",
			errorMsg:  "unexpected character",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create instance for this test
			var sv *structvar.StructVar
			if tt.setup != nil {
				sv = tt.setup()
			} else {
				sv = newTestStructVar()
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
	sv := &structvar.StructVar{
		Value: &TestObj{
			Name: "OldName",
		},
		Refs: map[string]string{
			"name": "${var.prefix} ${var.suffix}",
		},
	}

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

	sv := &structvar.StructVar{
		Value: &jobSettings,
		Refs: map[string]string{
			"tasks[0].run_job_task.job_id": "${resources.jobs.bar.id}",
		},
	}

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

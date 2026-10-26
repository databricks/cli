package structdiff_test

import (
	"testing"

	"github.com/databricks/cli/libs/structs/structdiff"
	"github.com/databricks/cli/libs/structs/structpath"
	"github.com/databricks/databricks-sdk-go/service/jobs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestKeyedPathUsesDiffSelectors(t *testing.T) {
	root := jobs.JobSettings{Tasks: []jobs.Task{
		{TaskKey: "second", DependsOn: []jobs.TaskDependency{{TaskKey: "first"}}},
		{TaskKey: "first"},
	}}
	keys := map[string]structdiff.KeyFunc{
		"tasks":               func(task jobs.Task) (string, string) { return "task_key", task.TaskKey },
		"tasks[*].depends_on": func(dependency jobs.TaskDependency) (string, string) { return "task_key", dependency.TaskKey },
	}
	for _, tt := range []struct{ path, want string }{
		{"tasks[0].depends_on[0].task_key", "tasks[task_key='second'].depends_on[task_key='first'].task_key"},
		{"tasks[1].task_key", "tasks[task_key='first'].task_key"},
		{"tasks[task_key='second'].depends_on[0].task_key", "tasks[task_key='second'].depends_on[task_key='first'].task_key"},
		{"tasks[task_key='first'].task_key", "tasks[task_key='first'].task_key"},
		{"tags['a.b']", "tags['a.b']"},
		{"", ""},
	} {
		t.Run(tt.path, func(t *testing.T) {
			result, err := structdiff.KeyedPath(&root, structpath.MustParsePath(tt.path), keys)
			require.NoError(t, err)
			assert.Equal(t, tt.want, result.String())
		})
	}
	path := structpath.MustParsePath("tasks[0].task_key")
	result, err := structdiff.KeyedPath(&root, path, nil)
	require.NoError(t, err)
	assert.Equal(t, path.String(), result.String())
}

func TestKeyedPathRejectsInvalidKeyFunctionsAndIndices(t *testing.T) {
	root := jobs.JobSettings{Tasks: []jobs.Task{{TaskKey: "main"}}}
	for _, tt := range []struct {
		name, path string
		keys       map[string]structdiff.KeyFunc
	}{
		{"invalid function", "tasks[0]", map[string]structdiff.KeyFunc{"tasks": 1}},
		{"wrong argument type", "tasks[0]", map[string]structdiff.KeyFunc{"tasks": func(int) (string, string) { return "", "" }}},
		{"missing element", "tasks[1]", map[string]structdiff.KeyFunc{
			"tasks":               func(task jobs.Task) (string, string) { return "task_key", task.TaskKey },
			"tasks[*].depends_on": func(dependency jobs.TaskDependency) (string, string) { return "task_key", dependency.TaskKey },
		}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			_, err := structdiff.KeyedPath(&root, structpath.MustParsePath(tt.path), tt.keys)
			assert.Error(t, err)
		})
	}
}

func TestKeyedPathRejectsEmptyOrAmbiguousIdentities(t *testing.T) {
	keys := map[string]structdiff.KeyFunc{
		"tasks": func(task jobs.Task) (string, string) { return "task_key", task.TaskKey },
	}
	for _, tt := range []struct {
		name, path string
		tasks      []jobs.Task
		want       string
	}{
		{"duplicate index", "tasks[1].new_cluster", []jobs.Task{{TaskKey: "main"}, {TaskKey: "main"}}, "ambiguous identity"},
		{"duplicate selector", "tasks[task_key='main'].new_cluster", []jobs.Task{{TaskKey: "main"}, {TaskKey: "main"}}, "ambiguous identity"},
		{"empty identity", "tasks[0].new_cluster", []jobs.Task{{}}, "empty identity"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			_, err := structdiff.KeyedPath(&jobs.JobSettings{Tasks: tt.tasks}, structpath.MustParsePath(tt.path), keys)
			require.ErrorContains(t, err, tt.want)
		})
	}
}

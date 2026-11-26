package validate

import (
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/databricks-sdk-go/service/compute"
	"github.com/databricks/databricks-sdk-go/service/jobs"
	"github.com/stretchr/testify/assert"
)

func TestIsInWheelTask(t *testing.T) {
	for path, want := range map[string]bool{
		"resources.jobs.a.tasks[0].python_wheel_task.parameters[0]": true,
		"resources.jobs.a.tasks[0].libraries[1].whl":                true,
		"resources.jobs.a.tasks[0].libraries[1].pypi.package":       false,
		"resources.jobs.a.tasks[0].notebook_task.base_parameters.x": false,
		"resources.jobs.a.name":                                     false,
		"resources.pipelines.a.libraries[0].whl":                    false,
	} {
		t.Run(path, func(t *testing.T) {
			assert.Equal(t, want, isInWheelTask(dyn.MustPathFromString(path)))
		})
	}
}

// The wrapper wraps only a task with a python_wheel_task and a workspace library; a local library
// may become one once the build uploads it.
func TestMayBeWrapped(t *testing.T) {
	wheel := &jobs.PythonWheelTask{PackageName: "p", EntryPoint: "e"}
	tasks := []jobs.Task{
		{TaskKey: "workspace_whl", PythonWheelTask: wheel, Libraries: []compute.Library{{Whl: "/Workspace/Shared/p.whl"}}},
		{TaskKey: "local_whl", PythonWheelTask: wheel, Libraries: []compute.Library{{Whl: "./dist/p.whl"}}},
		{TaskKey: "volumes_whl", PythonWheelTask: wheel, Libraries: []compute.Library{{Whl: "/Volumes/main/default/v/p.whl"}}},
		{TaskKey: "notebook", NotebookTask: &jobs.NotebookTask{NotebookPath: "/n"}, Libraries: []compute.Library{{Whl: "/Workspace/Shared/p.whl"}}},
	}
	b := &bundle.Bundle{Config: config.Root{Resources: config.Resources{Jobs: map[string]*resources.Job{
		"a": {JobSettings: jobs.JobSettings{Name: "a", Tasks: tasks}},
	}}}}

	for i, want := range []bool{true, true, false, false} {
		t.Run(tasks[i].TaskKey, func(t *testing.T) {
			path := dyn.NewPath(dyn.Key("resources"), dyn.Key("jobs"), dyn.Key("a"), dyn.Key("tasks"), dyn.Index(i), dyn.Key("python_wheel_task"))
			assert.Equal(t, want, mayBeWrapped(b, path))
		})
	}
}

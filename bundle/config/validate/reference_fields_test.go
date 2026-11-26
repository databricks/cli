package validate

import (
	"testing"

	"github.com/databricks/cli/libs/dyn"
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

package trampoline

import (
	"testing"

	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/dyn/convert"
	"github.com/databricks/databricks-sdk-go/service/compute"
	"github.com/databricks/databricks-sdk-go/service/jobs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// IsMovedTaskField must match exactly the fields CleanUp removes from a wrapped task: bundle validate
// skips references in them because the plan never sees them.
func TestIsMovedTaskFieldMatchesCleanUp(t *testing.T) {
	task := jobs.Task{
		TaskKey: "t",
		PythonWheelTask: &jobs.PythonWheelTask{
			PackageName:     "p",
			EntryPoint:      "e",
			Parameters:      []string{"a"},
			NamedParameters: map[string]string{"k": "v"},
		},
		NewCluster: &compute.ClusterSpec{SparkVersion: "15.4.x-scala2.12", NumWorkers: 1},
		// The wheel library is last, so removing it does not shift the others' indices.
		Libraries: []compute.Library{
			{Pypi: &compute.PythonPyPiLibrary{Package: "x"}},
			{Jar: "/Workspace/Shared/a.jar"},
			{Whl: "/Workspace/Shared/p.whl"},
		},
	}
	require.True(t, WrapsTask(task))

	before, err := convert.FromTyped(task, dyn.NilValue)
	require.NoError(t, err)
	cleaned := task
	require.NoError(t, (&pythonTrampoline{}).CleanUp(&cleaned))
	after, err := convert.FromTyped(cleaned, dyn.NilValue)
	require.NoError(t, err)

	leaves := 0
	err = dyn.WalkReadOnly(before, func(p dyn.Path, v dyn.Value) error {
		if v.Kind() == dyn.KindMap || v.Kind() == dyn.KindSequence {
			return nil
		}
		leaves++
		// A field CleanUp removed is missing from after, so the lookup fails.
		got, _ := dyn.GetByPath(after, p)
		removed := !got.IsValid() || got.AsAny() != v.AsAny()
		assert.Equal(t, removed, IsMovedTaskField(p), "field %s", p)
		return nil
	})
	require.NoError(t, err)
	assert.Greater(t, leaves, 5)
}

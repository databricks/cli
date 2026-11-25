package paths

import (
	"testing"

	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/databricks-sdk-go/service/apps"
	"github.com/databricks/databricks-sdk-go/service/compute"
	"github.com/databricks/databricks-sdk-go/service/jobs"
	"github.com/databricks/databricks-sdk-go/service/pipelines"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// collectVisitedPaths is a helper function that collects all visited paths for testing
func collectVisitedPaths(t *testing.T, root config.Root) []dyn.Path {
	return collectVisitedPathsOf(t, root, func(resources.LocalPath) bool { return true })
}

// collectVisitedPathsOf collects the visited paths of the fields for which keep returns true.
func collectVisitedPathsOf(t *testing.T, root config.Root, keep func(resources.LocalPath) bool) []dyn.Path {
	var actual []dyn.Path
	err := root.Mutate(func(value dyn.Value) (dyn.Value, error) {
		return VisitPaths(value, func(p dyn.Path, field resources.LocalPath, v dyn.Value) (dyn.Value, error) {
			if keep(field) {
				actual = append(actual, p)
			}
			return v, nil
		})
	})
	require.NoError(t, err)
	return actual
}

// The visit order decides which error is reported first, so it is pinned:
// resource types in canonical order, each in the order of its declarations,
// then artifacts.
func TestVisitPathsOrder(t *testing.T) {
	root := config.Root{
		Artifacts: config.Artifacts{
			"artifact0": &config.Artifact{Path: "foo"},
		},
		Resources: config.Resources{
			Apps: map[string]*resources.App{
				"app0": {App: apps.App{SourceCodePath: "foo"}},
			},
			Pipelines: map[string]*resources.Pipeline{
				"pipeline0": {CreatePipeline: pipelines.CreatePipeline{RootPath: "foo"}},
			},
			Jobs: map[string]*resources.Job{
				"job0": {JobSettings: jobs.JobSettings{Tasks: []jobs.Task{{
					TaskKey:      "task0",
					NotebookTask: &jobs.NotebookTask{NotebookPath: "foo"},
					Libraries:    []compute.Library{{Whl: "foo.whl"}},
				}}}},
			},
		},
	}

	actual := collectVisitedPaths(t, root)
	expected := []dyn.Path{
		dyn.MustPathFromString("resources.jobs.job0.tasks[0].notebook_task.notebook_path"),
		dyn.MustPathFromString("resources.jobs.job0.tasks[0].libraries[0].whl"),
		dyn.MustPathFromString("resources.pipelines.pipeline0.root_path"),
		dyn.MustPathFromString("resources.apps.app0.source_code_path"),
		dyn.MustPathFromString("artifacts.artifact0.path"),
	}

	assert.Equal(t, expected, actual)
}

// isLibraryPath reports whether a field holds a library, which may be outside the sync root.
func isLibraryPath(field resources.LocalPath) bool {
	return field.AllowOutsideSyncRoot
}

func isNotLibraryPath(field resources.LocalPath) bool {
	return !field.AllowOutsideSyncRoot
}

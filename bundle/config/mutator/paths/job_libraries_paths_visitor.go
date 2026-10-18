package paths

import (
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/libraries"
	"github.com/databricks/cli/libs/structs/structpath"
	"github.com/databricks/databricks-sdk-go/service/jobs"
)

func libraryRule(key string, field func(*jobs.Task, int) *string) taskRule {
	return taskRule{TranslateModeLocalRelative, nil, func(t *jobs.Task, c *taskCursor) error {
		for i := range t.Libraries {
			if err := c.yield(field(t, i), "libraries", i, key); err != nil {
				return err
			}
		}
		return nil
	}}
}

var jobTaskLibrariesRules = []taskRule{
	libraryRule("whl", func(t *jobs.Task, i int) *string { return &t.Libraries[i].Whl }),
	libraryRule("jar", func(t *jobs.Task, i int) *string { return &t.Libraries[i].Jar }),
}

func notLocalLibrary(s string) bool {
	return !libraries.IsLibraryLocal(s)
}

func noLocalPathInPipFlag(s string) bool {
	_, _, ok := libraries.IsLocalPathInPipFlag(s)
	return !ok
}

// VisitJobLibrariesPaths visits all libraries related paths in job resources and applies a function to each path.
func VisitJobLibrariesPaths(root *config.Root, fn VisitFunc) error {
	// Task libraries first, then environment dependencies (local libraries, then
	// local paths in pip flags), then the libraries of tasks nested in for_each_task.
	if err := visitTasksAt(root, false, jobTaskLibrariesRules, fn); err != nil {
		return err
	}
	for _, rule := range []struct {
		mode TranslateMode
		skip func(string) bool
	}{
		{TranslateModeLocalRelativeWithPrefix, notLocalLibrary},
		{TranslateModeEnvironmentPipFlag, noLocalPathInPipFlag},
	} {
		for _, name := range sortedKeys(root.Resources.Jobs) {
			job := root.Resources.Jobs[name]
			if job == nil {
				continue
			}
			for i, env := range job.Environments {
				if env.Spec == nil {
					continue
				}
				for j := range env.Spec.Dependencies {
					if err := visit(root, &env.Spec.Dependencies[j], rule.mode, rule.skip, fn, func() *structpath.PathNode {
						return structpath.NewPath(resourcePath("jobs", name), "environments", i, "spec", "dependencies", j)
					}); err != nil {
						return err
					}
				}
			}
		}
	}
	return visitTasksAt(root, true, jobTaskLibrariesRules, fn)
}

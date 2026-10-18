package paths

import (
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/libs/structs/structpath"
)

// VisitJobRunPaths visits local paths on job_runs so NormalizePaths can rewrite
// them relative to the bundle root. Not used by TranslatePaths: hashing still
// needs a local glob, not a workspace path.
func VisitJobRunPaths(root *config.Root, fn VisitFunc) error {
	for _, name := range sortedKeys(root.Resources.JobRuns) {
		r := root.Resources.JobRuns[name]
		if r == nil || r.Lifecycle == nil {
			continue
		}
		for i := range r.Lifecycle.Triggers {
			s := r.Lifecycle.Triggers[i].OnFileChange
			if s == nil {
				continue
			}
			if err := visit(root, s, TranslateModeLocalRelative, nil, fn, func() *structpath.PathNode {
				return structpath.NewPath(resourcePath("job_runs", name), "lifecycle", "triggers", i, "on_file_change")
			}); err != nil {
				return err
			}
		}
	}
	return nil
}

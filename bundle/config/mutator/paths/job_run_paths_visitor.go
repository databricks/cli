package paths

import (
	"github.com/databricks/cli/libs/structs/structpath"
	"github.com/databricks/cli/libs/structs/structvar"
)

// VisitJobRunPaths visits local paths on job_runs so NormalizePaths can rewrite
// them relative to the bundle root. Not used by TranslatePaths: hashing still
// needs a local glob, not a workspace path.
func VisitJobRunPaths(root structvar.View, fn VisitFunc) error {
	pattern := structpath.MustParsePattern("resources.job_runs.*.lifecycle.triggers[*].on_file_change")

	return structvar.ForEach(root, pattern, func(path *structpath.PathNode, value structvar.View) error {
		return fn(path, TranslateModeLocalRelative, value)
	})
}

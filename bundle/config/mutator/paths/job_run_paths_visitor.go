package paths

import (
	"github.com/databricks/cli/libs/dyn"
)

// VisitJobRunPaths visits local paths on job_runs so NormalizePaths can rewrite
// them relative to the bundle root. Not used by TranslatePaths: hashing still
// needs a local glob, not a workspace path.
func VisitJobRunPaths(value dyn.Value, fn VisitFunc) (dyn.Value, error) {
	pattern := dyn.NewPattern(
		dyn.Key("resources"),
		dyn.Key("job_runs"),
		dyn.AnyKey(),
		dyn.Key("lifecycle"),
		dyn.Key("triggers"),
		dyn.AnyIndex(),
		dyn.Key("on_file_change"),
	)

	return dyn.MapByPattern(value, pattern, func(path dyn.Path, value dyn.Value) (dyn.Value, error) {
		return fn(path, TranslateModeLocalRelative, value)
	})
}

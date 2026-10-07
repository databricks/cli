package paths

import (
	"github.com/databricks/cli/libs/structs/structpath"
	"github.com/databricks/cli/libs/structs/structvar"
)

// VisitFunc is called for every matching value. It may change the configuration
// at path (e.g. with [config.Root.Set]).
type VisitFunc func(path *structpath.PathNode, mode TranslateMode, value structvar.View) error

// VisitPaths visits all paths in bundle configuration
func VisitPaths(root structvar.View, fn VisitFunc) error {
	visitors := []func(structvar.View, VisitFunc) error{
		VisitJobPaths,
		VisitJobRunPaths,
		VisitJobLibrariesPaths,
		VisitAppPaths,
		VisitArtifactPaths,
		VisitAlertPaths,
		VisitDashboardPaths,
		VisitGenieSpacePaths,
		VisitPipelinePaths,
		VisitPipelineLibrariesPaths,
	}

	for _, visitor := range visitors {
		if err := visitor(root, fn); err != nil {
			return err
		}
	}

	return nil
}

// visitString calls fn for every string value matching pattern unless skip reports true for it.
// Values that are not strings are not visited.
func visitString(root structvar.View, pattern *structpath.PatternNode, mode TranslateMode, skip func(string) bool, fn VisitFunc) error {
	return structvar.ForEach(root, pattern, func(p *structpath.PathNode, v structvar.View) error {
		s, ok := v.AsString()
		if !ok || skip(s) {
			return nil
		}
		return fn(p, mode, v)
	})
}

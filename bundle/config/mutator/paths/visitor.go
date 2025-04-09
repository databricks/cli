package paths

import (
	"github.com/databricks/cli/libs/dyn"
)

type VisitFunc func(path dyn.Path, mode TranslateMode, value dyn.Value) (dyn.Value, error)

// VisitPaths visits all paths in bundle configuration
func VisitPaths(root dyn.Value, fn VisitFunc) (dyn.Value, error) {
	visitors := []func(dyn.Value, VisitFunc) (dyn.Value, error){
		VisitJobPaths,
		VisitAppPaths,
		VisitArtifactPaths,
		VisitDashboardPaths,
		VisitPipelinePaths,
	}

	newRoot := root
	for _, visitor := range visitors {
		updatedRoot, err := visitor(newRoot, fn)
		if err != nil {
			return dyn.InvalidValue, err
		}
		newRoot = updatedRoot
	}

	return newRoot, nil
}

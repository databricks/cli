package paths

import (
	"maps"
	"slices"

	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/libs/structs/structpath"
)

// VisitFunc is called for every path value. It may rewrite the value through the
// pointer; the value keeps its locations.
type VisitFunc func(path *structpath.PathNode, mode TranslateMode, value *string) error

// VisitPaths visits all paths in bundle configuration
func VisitPaths(root *config.Root, fn VisitFunc) error {
	visitors := []func(*config.Root, VisitFunc) error{
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

// visit calls fn for the path value *s unless skip reports true for it or it is empty
// and not written in the source (an explicit "" is translated too). The path is only
// built for values that are visited or empty.
func visit(root *config.Root, s *string, mode TranslateMode, skip func(string) bool, fn VisitFunc, path func() *structpath.PathNode) error {
	if skip != nil && skip(*s) {
		return nil
	}
	if *s != "" {
		return fn(path(), mode, s)
	}
	p := path()
	if len(root.ExactLocations(p)) == 0 {
		return nil
	}
	return fn(p, mode, s)
}

// sortedKeys returns the keys of a resource map in the order the paths are visited.
func sortedKeys[V any](m map[string]V) []string {
	return slices.Sorted(maps.Keys(m))
}

// resourcePath returns the path of the resource name of the given type.
func resourcePath(typ, name string) *structpath.PathNode {
	return structpath.NewPath(nil, "resources", typ, name)
}

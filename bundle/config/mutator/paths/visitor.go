package paths

import (
	"maps"
	"reflect"
	"slices"

	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/libs/structs/structpath"
	"github.com/databricks/cli/libs/structs/structvar"
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

// visit calls fn for the path value *s, the field key of the struct owner, unless skip
// reports true for it or it is not present: an empty value is present only if it is in
// the owner's ForceSendFields (as [structvar.View] decides). Elements of lists have no
// owner and are always present. The path is only built for values that are visited.
func visit(owner any, key string, s *string, mode TranslateMode, skip func(string) bool, fn VisitFunc, path func() *structpath.PathNode) error {
	if skip != nil && skip(*s) {
		return nil
	}
	if *s == "" && owner != nil && !isForced(owner, key) {
		return nil
	}
	return fn(path(), mode, s)
}

// isForced reports whether the field key of the struct *owner is in its ForceSendFields.
// It is only called for empty values.
func isForced(owner any, key string) bool {
	v := reflect.ValueOf(owner).Elem()
	return structvar.GetStructInfo(v.Type()).IsForceSend(v, key)
}

// sortedKeys returns the keys of a resource map in the order the paths are visited.
func sortedKeys[V any](m map[string]V) []string {
	return slices.Sorted(maps.Keys(m))
}

// resourcePath returns the path of the resource name of the given type.
func resourcePath(typ, name string) *structpath.PathNode {
	return structpath.NewPath(nil, "resources", typ, name)
}

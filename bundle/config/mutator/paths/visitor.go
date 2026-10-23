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

// visitArgs describes one path value for [visit].
type visitArgs struct {
	// owner is the struct that holds the field, for its ForceSendFields; nil for list
	// elements, which are always present.
	owner any

	// field is the JSON key of the field in owner.
	field string

	// value is the path value; fn may rewrite it.
	value *string

	mode TranslateMode

	// path is the path of the value.
	path *structpath.PathNode
}

// present reports whether the path value is present: an empty value is present only if
// it is in the owner's ForceSendFields (as [structvar.View] decides). Callers that visit
// many empty values check it before building the path.
func present(owner any, field string, value *string) bool {
	return *value != "" || owner == nil || isForced(owner, field)
}

// visit calls fn for the path value described by a, unless it is not present.
func visit(fn VisitFunc, a visitArgs) error {
	if !present(a.owner, a.field, a.value) {
		return nil
	}
	return fn(a.path, a.mode, a.value)
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

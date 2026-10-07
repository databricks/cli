package dyn

import "github.com/databricks/cli/libs/structs/structpath"

// ToStructPath converts a [Path] to a [structpath.PathNode].
func ToStructPath(p Path) *structpath.PathNode {
	var result *structpath.PathNode
	for _, c := range p {
		if c.isKey() {
			result = structpath.NewStringKey(result, c.key)
		} else {
			result = structpath.NewIndex(result, c.index)
		}
	}
	return result
}

// ToStructPaths converts each of paths with [ToStructPath], e.g. for diag.Diagnostic.Paths.
func ToStructPaths(paths ...Path) []*structpath.PathNode {
	out := make([]*structpath.PathNode, len(paths))
	for i, p := range paths {
		out[i] = ToStructPath(p)
	}
	return out
}

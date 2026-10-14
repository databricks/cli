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

// FromStructPath converts a [structpath.PathNode] of keys and indices to a [Path].
// It reports false for nodes a [Path] cannot represent (key-value selectors, wildcards).
func FromStructPath(p *structpath.PathNode) (Path, bool) {
	var out Path
	for _, n := range p.AsSlice() {
		if k, ok := n.StringKey(); ok {
			out = append(out, Key(k))
		} else if i, ok := n.Index(); ok {
			out = append(out, Index(i))
		} else {
			return nil, false
		}
	}
	return out, true
}

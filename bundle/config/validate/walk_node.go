package validate

import (
	"fmt"
	"slices"

	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/dyn/dynvar"
	"github.com/databricks/cli/libs/structs/structpath"
)

// This file is the bridge between the generated typed walkers (required_walk.go,
// enum_walk.go) and the dynamic configuration. The typed configuration cannot tell an
// unset field from one that is null, set to its zero value or to a pure variable
// reference, and it has no locations, so a node follows the [dyn.Value] at its path
// for those. The typed configuration is derived from the dynamic one after every
// mutation, so the two agree. Replace the node once the configuration no longer uses
// dyn.

// node is a position in the configuration visited by a generated walker. Nodes live
// on the stack and link to their parent; the path is only built for a diagnostic.
type node struct {
	parent *node
	key    string
	index  int // -1 for a key, -2 for the root
	v      dyn.Value
}

func rootNode(v dyn.Value) node {
	return node{index: -2, v: v}
}

// child returns the node of the key k of n.
func (n *node) child(k string) node {
	return node{parent: n, key: k, index: -1, v: n.v.Get(k)}
}

// elem returns the node of element i of n.
func (n *node) elem(i int) node {
	return node{parent: n, index: i, v: n.v.Index(i)}
}

// exists reports whether the configuration has a value (possibly null) at n.
func (n *node) exists() bool {
	return n.v.IsValid()
}

// has reports whether the key k of n is set to a value that is not null.
func (n *node) has(k string) bool {
	kind := n.v.Get(k).Kind()
	return kind != dyn.KindInvalid && kind != dyn.KindNil
}

func (n *node) path() *structpath.PathNode {
	switch n.index {
	case -2:
		return nil
	case -1:
		return structpath.NewStringKey(n.parent.path(), n.key)
	default:
		return structpath.NewIndex(n.parent.path(), n.index)
	}
}

// walker holds the diagnostics of a generated walk.
type walker struct {
	diags diag.Diagnostics
}

// missing reports that the required field of n is not set. A pure reference
// (e.g. "${resources.jobs.base.schedule}") has no fields of its own; the referenced
// value is checked where it is defined.
func (w *walker) missing(n *node, field string) {
	if s, ok := n.v.AsString(); ok && dynvar.IsPureVariableReference(s) {
		return
	}
	w.diags = append(w.diags, diag.Diagnostic{
		Severity:  diag.Warning,
		Summary:   fmt.Sprintf("required field %q is not set", field),
		Locations: n.v.Locations(),
		Paths:     []*structpath.PathNode{n.path()},
	})
}

// enumField checks the enum field k of n, whose typed value is s.
func (w *walker) enumField(n *node, k, s string, valid []string) {
	if s != "" && slices.Contains(valid, s) {
		return
	}
	c := n.child(k)
	w.enumValue(&c, valid)
}

// enumElem checks the enum element n, whose typed value is s.
func (w *walker) enumElem(n *node, s string, valid []string) {
	if s != "" && slices.Contains(valid, s) {
		return
	}
	w.enumValue(n, valid)
}

// enumValue reports the value of n if it is a string that is not one of valid. Values
// with variable references are not resolved yet and are skipped.
func (w *walker) enumValue(n *node, valid []string) {
	s, ok := n.v.AsString()
	if !ok || dynvar.ContainsVariableReference(s) || slices.Contains(valid, s) {
		return
	}
	w.diags = append(w.diags, diag.Diagnostic{
		Severity:  diag.Warning,
		Summary:   fmt.Sprintf("invalid value %q for enum field. Valid values are %v", s, valid),
		Locations: n.v.Locations(),
		Paths:     []*structpath.PathNode{n.path()},
	})
}

// sortByWalkOrder orders diagnostics with one path each in the order [dyn.WalkReadOnly]
// visits their paths in root, which the walkers do not follow.
func sortByWalkOrder(root dyn.Value, diags diag.Diagnostics) {
	type item struct {
		pos []int
		d   diag.Diagnostic
	}
	items := make([]item, len(diags))
	for i, d := range diags {
		items[i] = item{walkPosition(root, d.Paths[0]), d}
	}
	slices.SortStableFunc(items, func(a, b item) int {
		return slices.Compare(a.pos, b.pos)
	})
	for i := range items {
		diags[i] = items[i].d
	}
}

// walkPosition returns the position of each component of path p in root: the index of
// a key among the keys of its map, or the index of an element.
func walkPosition(root dyn.Value, p *structpath.PathNode) []int {
	v := root
	var pos []int
	for _, c := range p.AsSlice() {
		if i, ok := c.Index(); ok {
			pos = append(pos, i)
			v = v.Index(i)
			continue
		}
		k, _ := c.StringKey()
		m, _ := v.AsMap()
		i := slices.IndexFunc(m.Pairs(), func(pair dyn.Pair) bool {
			return pair.Key.MustString() == k
		})
		pos = append(pos, i)
		v = v.Get(k)
	}
	return pos
}

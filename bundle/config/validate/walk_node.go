package validate

import (
	"fmt"
	"reflect"
	"slices"

	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/structs/structpath"
	"github.com/databricks/cli/libs/structs/structvar"
)

// node is a position in the configuration visited by the generated walkers
// (required_walk.go, enum_walk.go). Nodes live on the stack and link to their parent;
// the path is only built when a diagnostic needs it. A node also follows the
// locations tree and the tree of pure references, so neither needs a path lookup.
type node struct {
	parent *node
	key    string
	index  int // -1 for a key, -2 for the root
	locs   *structvar.Locations
	refs   *refNode
}

func (n *node) child(k string) node {
	return node{parent: n, key: k, index: -1, locs: n.locs.Key(k), refs: n.refs.key(k)}
}

func (n *node) elem(i int) node {
	return node{parent: n, index: i, locs: n.locs.Index(i), refs: n.refs.elem(i)}
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

// refNode is a tree of the paths that hold pure variable references.
type refNode struct {
	ref   bool
	keys  map[string]*refNode
	elems map[int]*refNode
}

func (r *refNode) key(k string) *refNode {
	if r == nil {
		return nil
	}
	return r.keys[k]
}

func (r *refNode) elem(i int) *refNode {
	if r == nil {
		return nil
	}
	return r.elems[i]
}

func newRefTree(root *config.Root) *refNode {
	var out *refNode
	for p := range root.References() {
		if out == nil {
			out = &refNode{}
		}
		n := out
		for _, c := range p.AsSlice() {
			if k, ok := c.StringKey(); ok {
				if n.keys == nil {
					n.keys = map[string]*refNode{}
				}
				if n.keys[k] == nil {
					n.keys[k] = &refNode{}
				}
				n = n.keys[k]
			} else if i, ok := c.Index(); ok {
				if n.elems == nil {
					n.elems = map[int]*refNode{}
				}
				if n.elems[i] == nil {
					n.elems[i] = &refNode{}
				}
				n = n.elems[i]
			}
		}
		n.ref = true
	}
	return out
}

// walker holds the state of a generated walk.
type walker struct {
	diags diag.Diagnostics
}

// walk runs the generated walker fn over the configuration of root.
func walk(root *config.Root, fn func(*walker, *node, *config.Root)) diag.Diagnostics {
	w := &walker{}
	n := node{index: -2, locs: root.View().Locs(), refs: newRefTree(root)}
	fn(w, &n, root)
	return w.diags
}

// isRef reports whether the field k of n holds a pure variable reference. Only fields
// that cannot hold a string can.
func (w *walker) isRef(n *node, k string) bool {
	r := n.refs.key(k)
	return r != nil && r.ref
}

// forced reports whether name is listed in the ForceSendFields fsf.
func forced(fsf []string, name string) bool {
	return slices.Contains(fsf, name)
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
		Locations: n.locs.Get(),
		Paths:     []*structpath.PathNode{n.path()},
	})
}

// enumValue reports s, the value of n, if it is not one of valid. Values with variable
// references are not resolved yet and are skipped.
func (w *walker) enumValue(n *node, s string, valid []string) {
	if slices.Contains(valid, s) || structvar.ContainsVariableReference(s) {
		return
	}
	w.diags = append(w.diags, diag.Diagnostic{
		Severity:  diag.Warning,
		Summary:   fmt.Sprintf("invalid value %q for enum field. Valid values are %v", s, valid),
		Locations: n.locs.Get(),
		Paths:     []*structpath.PathNode{n.path()},
	})
}

// isZeroStruct reports whether the struct *v is the zero value. Only the few struct
// fields that are not pointers need it.
func isZeroStruct[T any](v *T) bool {
	return reflect.ValueOf(v).Elem().IsZero()
}

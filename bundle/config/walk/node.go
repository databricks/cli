// Package walk holds the node type of the generated typed walkers over the bundle
// configuration and the walkers that are not specific to one mutator.
package walk

import (
	"reflect"
	"slices"

	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/structs/structpath"
	"github.com/databricks/cli/libs/structs/structvar"
)

// Node is a position in the configuration visited by a generated walker. Nodes live
// on the stack and link to their parent; the path is only built when needed (e.g. for
// a diagnostic). A node also follows the locations tree and the tree of pure
// references, so neither needs a path lookup.
type Node struct {
	parent *Node
	key    string
	index  int // -1 for a key, -2 for the root
	locs   *structvar.Locations
	refs   *refNode
}

// Root returns the root node of the configuration r. Its nodes do not follow the pure
// references: use [RootWithReferences] for [Node.IsRef].
func Root(r *config.Root) Node {
	return Node{index: -2, locs: r.LocationTree()}
}

// RootWithReferences is [Root] whose nodes also follow the pure references.
func RootWithReferences(r *config.Root) Node {
	n := Root(r)
	n.refs = newRefTree(r)
	return n
}

// Child returns the node of the key k of n.
func (n *Node) Child(k string) Node {
	return Node{parent: n, key: k, index: -1, locs: n.locs.Key(k), refs: n.refs.key(k)}
}

// Elem returns the node of element i of n.
func (n *Node) Elem(i int) Node {
	return Node{parent: n, index: i, locs: n.locs.Index(i), refs: n.refs.elem(i)}
}

// Path returns the path of n.
func (n *Node) Path() *structpath.PathNode {
	switch n.index {
	case -2:
		return nil
	case -1:
		return structpath.NewStringKey(n.parent.Path(), n.key)
	default:
		return structpath.NewIndex(n.parent.Path(), n.index)
	}
}

// Locations returns the locations of the value at n.
func (n *Node) Locations() *structvar.Locations {
	return n.locs
}

// Location returns the first location of the value at n.
func (n *Node) Location() diag.Location {
	locs := n.locs.Get()
	if len(locs) == 0 {
		return diag.Location{}
	}
	return locs[0]
}

// IsRef reports whether the key k of n holds a pure variable reference. Only fields
// that cannot hold a string can.
func (n *Node) IsRef(k string) bool {
	r := n.refs.key(k)
	return r != nil && r.ref
}

// HasRefs reports whether the key k of n holds a pure variable reference or has one
// below it (e.g. a struct whose only set field is a reference).
func (n *Node) HasRefs(k string) bool {
	return n.refs.key(k) != nil
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

// Forced reports whether name is listed in the ForceSendFields fsf. Generated walkers
// use it to decide whether a zero value is set.
func Forced(fsf []string, name string) bool {
	return slices.Contains(fsf, name)
}

// IsZeroStruct reports whether the struct *v is the zero value. Generated walkers only
// need it for the few struct fields that are not pointers.
func IsZeroStruct[T any](v *T) bool {
	return reflect.ValueOf(v).Elem().IsZero()
}

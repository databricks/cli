package structvar

import (
	"maps"
	"slices"

	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/structs/structpath"
)

// Locations records the source locations of the values of a typed value, as a tree
// that follows the value: one node per map key or sequence index.
//
// Locations is immutable: operations that change it return a new tree that shares
// unchanged subtrees, so copies of a struct holding one can share it safely.
// A nil *Locations is an empty tree.
type Locations struct {
	locs  []diag.Location
	keys  map[string]*Locations
	elems []*Locations
}

// pathComponent is a key or an index of a path.
type pathComponent struct {
	key   string
	index int
	isKey bool
}

// components returns the keys and indices of p.
func components(p *structpath.PathNode) []pathComponent {
	var out []pathComponent
	for _, n := range p.AsSlice() {
		if k, ok := n.StringKey(); ok {
			out = append(out, pathComponent{key: k, isKey: true})
		} else if i, ok := n.Index(); ok {
			out = append(out, pathComponent{index: i})
		}
	}
	return out
}

func (l *Locations) child(c pathComponent) *Locations {
	if l == nil {
		return nil
	}
	if c.isKey {
		return l.keys[c.key]
	}
	if c.index < 0 || c.index >= len(l.elems) {
		return nil
	}
	return l.elems[c.index]
}

// Key returns the locations of map key or struct field k.
func (l *Locations) Key(k string) *Locations {
	return l.child(pathComponent{key: k, isKey: true})
}

// Index returns the locations of sequence element i.
func (l *Locations) Index(i int) *Locations {
	return l.child(pathComponent{index: i})
}

// Get returns the locations recorded for the value itself.
func (l *Locations) Get() []diag.Location {
	if l == nil {
		return nil
	}
	return slices.Clone(l.locs)
}

// At returns the locations of the value at path. Values without recorded locations
// (e.g. set by code) have none.
func (l *Locations) At(path *structpath.PathNode) []diag.Location {
	return l.lookup(components(path)).Get()
}

// Nearest returns the locations of the value at path or, if it has none, of the
// closest ancestor that has some. Use it to point diagnostics at the most specific
// place there is, not to decide where a value was defined.
func (l *Locations) Nearest(path *structpath.PathNode) []diag.Location {
	var nearest []diag.Location
	n := l
	for _, c := range components(path) {
		if n == nil {
			break
		}
		if len(n.locs) > 0 {
			nearest = n.locs
		}
		n = n.child(c)
	}
	if n != nil && len(n.locs) > 0 {
		nearest = n.locs
	}
	return slices.Clone(nearest)
}

func (l *Locations) lookup(path []pathComponent) *Locations {
	n := l
	for _, c := range path {
		n = n.child(c)
		if n == nil {
			return nil
		}
	}
	return n
}

// Sub returns the locations of the value at path and below.
func (l *Locations) Sub(path *structpath.PathNode) *Locations {
	return l.lookup(components(path))
}

// With returns a copy of l where the locations of the value at path and below are sub.
func (l *Locations) With(path *structpath.PathNode, sub *Locations) *Locations {
	return l.with(components(path), sub)
}

func (l *Locations) with(path []pathComponent, sub *Locations) *Locations {
	if len(path) == 0 {
		return sub
	}
	child := l.child(path[0]).with(path[1:], sub)
	return l.withChild(path[0], child)
}

func (l *Locations) clone() *Locations {
	if l == nil {
		return &Locations{}
	}
	c := *l
	c.keys = maps.Clone(l.keys)
	c.elems = slices.Clone(l.elems)
	return &c
}

func (l *Locations) withChild(c pathComponent, child *Locations) *Locations {
	out := l.clone()
	out.setChild(c, child)
	return out
}

// setChild sets the child c of l in place. Only use it on a node the caller owns,
// e.g. one just returned by [Locations.WithLocations], which builds a fresh node.
func (l *Locations) setChild(c pathComponent, child *Locations) {
	if c.isKey {
		if child == nil {
			delete(l.keys, c.key)
			return
		}
		if l.keys == nil {
			l.keys = map[string]*Locations{}
		}
		l.keys[c.key] = child
		return
	}
	if c.index < 0 {
		return
	}
	for len(l.elems) <= c.index {
		l.elems = append(l.elems, nil)
	}
	l.elems[c.index] = child
}

// WithLocations returns a copy of l where the value itself has the locations locs.
func (l *Locations) WithLocations(locs []diag.Location) *Locations {
	c := l.clone()
	c.locs = slices.Clone(locs)
	return c
}

// WithSequence returns a copy of l where the elements of the sequence at path are
// rebuilt from the old ones: new element i has the locations of old elements sources[i]
// merged (see [MergeLocations]). Use it after reordering, merging, splitting or
// dropping sequence elements.
func (l *Locations) WithSequence(path *structpath.PathNode, sources [][]int) *Locations {
	p := components(path)
	n := l.lookup(p)
	if n == nil {
		return l
	}
	c := n.clone()
	c.elems = make([]*Locations, len(sources))
	for i, src := range sources {
		var merged *Locations
		for j, k := range src {
			if j == 0 {
				merged = n.Index(k)
				continue
			}
			merged = MergeLocations(merged, n.Index(k))
		}
		c.elems[i] = merged
	}
	return l.with(p, c)
}

// MergeLocations returns the locations of the value obtained by merging the value
// described by b into the one described by a, following merge.Merge: maps and
// sequences keep a's locations and accumulate b's, other values take b's and
// accumulate a's.
func MergeLocations(a, b *Locations) *Locations {
	switch {
	case a == nil:
		return b
	case b == nil:
		return a
	}
	isComposite := len(a.keys) > 0 || len(a.elems) > 0 || len(b.keys) > 0 || len(b.elems) > 0
	if !isComposite {
		return b.WithLocations(concatLocations(b.locs, a.locs))
	}
	c := a.clone()
	c.locs = concatLocations(a.locs, b.locs)
	for k, bv := range b.keys {
		if c.keys == nil {
			c.keys = map[string]*Locations{}
		}
		c.keys[k] = MergeLocations(a.keys[k], bv)
	}
	c.elems = append(slices.Clone(a.elems), b.elems...)
	return c
}

func concatLocations(a, b []diag.Location) []diag.Location {
	if len(b) == 0 {
		return a
	}
	return append(slices.Clone(a), b...)
}

// Map returns a copy of l where the locations of every value are replaced by fn(path, locations).
func (l *Locations) Map(fn func(path *structpath.PathNode, locs []diag.Location) []diag.Location) *Locations {
	return l.mapLocations(nil, fn)
}

func (l *Locations) mapLocations(path *structpath.PathNode, fn func(*structpath.PathNode, []diag.Location) []diag.Location) *Locations {
	if l == nil {
		return nil
	}
	c := l.clone()
	c.locs = fn(path, slices.Clone(l.locs))
	for k, v := range c.keys {
		c.keys[k] = v.mapLocations(structpath.NewStringKey(path, k), fn)
	}
	for i, v := range c.elems {
		c.elems[i] = v.mapLocations(structpath.NewIndex(path, i), fn)
	}
	return c
}

// child returns the path of this component below prefix.
func (c pathComponent) child(prefix *structpath.PathNode) *structpath.PathNode {
	if c.isKey {
		return structpath.NewStringKey(prefix, c.key)
	}
	return structpath.NewIndex(prefix, c.index)
}

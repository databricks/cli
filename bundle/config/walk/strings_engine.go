package walk

import (
	"reflect"
	"slices"
	"strings"
	"unsafe"

	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/variable"
	"github.com/databricks/cli/libs/structs/structinfo"
)

var (
	resourcesInfo = structinfo.For(reflect.TypeFor[config.Resources]())
	lookupInfo    = structinfo.For(reflect.TypeFor[variable.Lookup]())
)

// strct visits the strings of the struct t at p, in field order. A value replaced by a
// pure reference hides what is below it; the reference itself is in
// config.Root.References.
func (c *stringsWalk) strct(t *structinfo.Type, n *Node, p unsafe.Pointer) {
	if n.isRef() {
		return
	}
	ops := t.Walk[ruleStrings]
	for i := range ops {
		c.field(t, n, p, &ops[i])
	}
}

// ruleStrings is the index of the strings rule in structinfo.Type.Walk.
const ruleStrings = 2

// field visits the strings of the field that op describes in the struct t at p.
func (c *stringsWalk) field(t *structinfo.Type, n *Node, p unsafe.Pointer, op *structinfo.Op) {
	if t == rootInfo && c.skipResources && op.Key == "resources" {
		return
	}
	var fp unsafe.Pointer
	if op.Slow {
		fp = t.Fields[op.Idx].Loc.Resolve(p)
		if fp == nil {
			return
		}
	} else {
		fp = unsafe.Add(p, op.Offset)
	}
	if op.Kind == structinfo.KindString {
		// The common case: a string field.
		s := *(*string)(fp)
		if c.mode == EmptyFields && s == "" && !op.NoOmitEmpty && t.ForcedOp(p, op) {
			ch := c.child(n, op.Key)
			c.fn(ch, "")
			c.pop()
		}
		if c.match(s) {
			ch := c.child(n, op.Key)
			if r, ok := c.fn(ch, s); ok {
				*(*string)(fp) = r
			}
			c.pop()
		}
		return
	}
	if op.Kind != structinfo.KindInterface && n.IsRef(op.Key) {
		// A field replaced by a pure reference hides what is below it.
		return
	}
	if c.mode == EmptyFields {
		c.emptyField(t, n, p, op, fp)
	}
	c.visit(op.Elem, n, op.Key, fp)
}

// emptyField visits the field in mode EmptyFields if it holds an empty string that is
// present: a forced string, a non-nil pointer to a string, or an interface. Fields
// without omitempty are skipped: their zero value is always sent.
func (c *stringsWalk) emptyField(t *structinfo.Type, n *Node, p unsafe.Pointer, op *structinfo.Op, fp unsafe.Pointer) {
	if op.NoOmitEmpty {
		return
	}
	switch {
	case op.Kind == structinfo.KindString:
		if *(*string)(fp) != "" || !t.ForcedOp(p, op) {
			return
		}
	case op.Kind == structinfo.KindPtr && op.Elem.Elem.Kind == structinfo.KindString:
		q := *(*unsafe.Pointer)(fp)
		if q == nil || *(*string)(q) != "" {
			return
		}
	case op.Kind == structinfo.KindInterface:
		if s, ok := (*(*any)(fp)).(string); !ok || s != "" {
			return
		}
	default:
		return
	}
	ch := c.child(n, op.Key)
	c.fn(ch, "")
	c.pop()
}

// visit visits the strings of the value of type t at p, the field or key k of n.
func (c *stringsWalk) visit(t *structinfo.Type, n *Node, k string, p unsafe.Pointer) {
	switch t.Kind {
	case structinfo.KindString:
		if s := *(*string)(p); c.match(s) {
			ch := c.child(n, k)
			if r, ok := c.fn(ch, s); ok {
				*(*string)(p) = r
			}
			c.pop()
		}
	case structinfo.KindInterface:
		if a := (*any)(p); *a != nil {
			ch := c.child(n, k)
			c.any(ch, a)
			c.pop()
		}
	case structinfo.KindPtr:
		if q := *(*unsafe.Pointer)(p); q != nil {
			if t.Elem.Kind == structinfo.KindStruct {
				ch := c.child(n, k)
				c.strct(t.Elem, ch, q)
				c.pop()
			} else {
				c.visit(t.Elem, n, k, q)
			}
		}
	case structinfo.KindStruct:
		ch := c.child(n, k)
		c.strct(t, ch, p)
		c.pop()
	case structinfo.KindSlice:
		h := (*sliceHeader)(p)
		if h.len == 0 {
			return
		}
		cn := c.child(n, k)
		for i := range h.len {
			c.elem(t.Elem, cn, i, unsafe.Add(h.data, uintptr(i)*t.Elem.Size))
		}
		c.pop()
	case structinfo.KindMap:
		c.mapValue(t, n, k, p)
	default:
	}
}

// elem is [stringsWalk.visit] for the element i of a slice.
func (c *stringsWalk) elem(t *structinfo.Type, cn *Node, i int, p unsafe.Pointer) {
	ch := c.elemNode(cn, i)
	// The element node is the node of the value: visit it in place.
	switch t.Kind {
	case structinfo.KindString:
		if s := *(*string)(p); c.match(s) {
			if r, ok := c.fn(ch, s); ok {
				*(*string)(p) = r
			}
		}
	case structinfo.KindInterface:
		if a := (*any)(p); *a != nil {
			c.any(ch, a)
		}
	case structinfo.KindPtr:
		if q := *(*unsafe.Pointer)(p); q != nil {
			c.target(t.Elem, ch, q)
		}
	default:
		c.target(t, ch, p)
	}
	c.pop()
}

// target visits the strings of the value of type t at p whose node n is already made.
func (c *stringsWalk) target(t *structinfo.Type, n *Node, p unsafe.Pointer) {
	switch t.Kind {
	case structinfo.KindStruct:
		c.strct(t, n, p)
	case structinfo.KindString:
		if s := *(*string)(p); c.match(s) {
			if r, ok := c.fn(n, s); ok {
				*(*string)(p) = r
			}
		}
	case structinfo.KindInterface:
		if a := (*any)(p); *a != nil {
			c.any(n, a)
		}
	case structinfo.KindPtr:
		if q := *(*unsafe.Pointer)(p); q != nil {
			c.target(t.Elem, n, q)
		}
	case structinfo.KindSlice:
		h := (*sliceHeader)(p)
		for i := range h.len {
			c.elem(t.Elem, n, i, unsafe.Add(h.data, uintptr(i)*t.Elem.Size))
		}
	case structinfo.KindMap:
		c.mapElems(t, n, p)
	default:
	}
}

// mapValue visits the map t at p, the field or key k of n, in the order of its keys.
func (c *stringsWalk) mapValue(t *structinfo.Type, n *Node, k string, p unsafe.Pointer) {
	if *(*unsafe.Pointer)(p) == nil {
		return
	}
	cn := c.child(n, k)
	c.mapElems(t, cn, p)
	c.pop()
}

// mapElems visits the elements of the map t at p, whose node is cn.
func (c *stringsWalk) mapElems(t *structinfo.Type, cn *Node, p unsafe.Pointer) {
	switch t.MapKind {
	case structinfo.MapStringPtr:
		m := *(*map[string]unsafe.Pointer)(p)
		for _, k := range sortedKeys(m) {
			if q := m[k]; q != nil {
				ch := c.child(cn, k)
				c.target(t.Elem.Elem, ch, q)
				c.pop()
			}
		}
		return
	case structinfo.MapStringString:
		m := *(*map[string]string)(p)
		for _, k := range sortedKeys(m) {
			if s := m[k]; c.match(s) {
				ch := c.child(cn, k)
				if r, ok := c.fn(ch, s); ok {
					m[k] = r
				}
				c.pop()
			}
		}
		return
	case structinfo.MapOther:
	}
	m := reflect.NewAt(t.Typ, p).Elem()
	if m.Len() == 0 {
		return
	}
	keys := m.MapKeys()
	slices.SortFunc(keys, func(a, b reflect.Value) int { return strings.Compare(a.String(), b.String()) })
	et := t.Elem
	for _, key := range keys {
		ch := c.child(cn, key.String())
		switch et.Kind {
		case structinfo.KindPtr:
			// The pointer is not changed: visit what it points to.
			if q := m.MapIndex(key).UnsafePointer(); q != nil {
				c.target(et.Elem, ch, q)
			}
		case structinfo.KindString:
			// Map values are not addressable: write the replacement back.
			if s := m.MapIndex(key).String(); c.match(s) {
				if r, ok := c.fn(ch, s); ok {
					m.SetMapIndex(key, reflect.ValueOf(r).Convert(et.Typ))
				}
			}
		default:
			cp := reflect.New(et.Typ)
			cp.Elem().Set(m.MapIndex(key))
			c.target(et, ch, cp.UnsafePointer())
			m.SetMapIndex(key, cp.Elem())
		}
		c.pop()
	}
}

// resource visits the strings of the resource name of type typ, whose node rn is the
// node of "resources".
func (c *stringsWalk) resource(rn *Node, r *config.Resources, typ, name string) {
	for i := range resourcesInfo.Fields {
		f := &resourcesInfo.Fields[i]
		if f.Key != typ {
			continue
		}
		m := reflect.NewAt(f.Typ, f.Loc.Resolve(unsafe.Pointer(r))).Elem()
		q := m.MapIndex(reflect.ValueOf(name).Convert(f.Typ.Key())).UnsafePointer()
		if q == nil || f.Type.Elem.Elem.Has&structinfo.RuleStrings == 0 {
			return
		}
		ch := rn.Child(typ)
		en := ch.Child(name)
		c.strct(f.Type.Elem.Elem, c.root(en), q)
		return
	}
}

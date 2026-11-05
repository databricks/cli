package walk

import (
	"reflect"
	"unsafe"

	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/libs/structs/structinfo"
	"github.com/databricks/databricks-sdk-go/service/jobs"
)

// TODO: possible optimizations, in the order to try them. The enum walk is 1.5x slower than
// the generated walker it replaced (0.62 ms vs 0.42 ms for 1000 jobs, about 4 ns vs 2.6 ns
// per visited field); the profile shows the time in field() itself, not in allocation.
//   - Split each walk list by kind (nil-able pointers, enum strings, containers) and loop
//     over each list separately: the shared switch on op.Kind is data-dependent and hard
//     to predict, while the generated code has one specialised branch per field.
//   - Unroll the recursion: copy the walk entries of a struct held by value into the
//     parent's list with the offset added, and guard the entries of a pointer struct with
//     one nil check, so a call and a loop setup per nested struct go away. Recursive types
//     (jobs.Task -> ForEachTask -> Task) keep real recursion at the recursive edge.
//   - The strings walk (strings_engine.go) is 1.3x slower than the generated one it replaced
//     (2.9 ms vs 2.2 ms for 1000 jobs, 24k allocs vs 76k): the time is spread over the
//     prologue of field(), the "${" search and the IsRef lookups, with no single hot spot.
//     Splitting the cold paths (EmptyFields, interface and map values) out of field() to
//     shrink its frame is the first thing to try.
//   - Keep the node arena on the Go stack by passing nodes by value to callbacks that
//     do not retain them (avoids write barriers on the heap arena).
//
// ruleWalk walks the configuration structs by the cached [structinfo.Type] of each struct:
// a field is reached by adding an offset to the address of its struct, and only the
// fields that lead to something the rule looks at are visited.
type ruleWalk struct {
	arena

	rule    int // index into structinfo.Type.Walk
	missing func(n *Node, field string)
	enum    func(n *Node, s string, valid []string)
}

// maxDepth is the number of nodes between the root and the deepest value.
const maxDepth = 64

// arena holds the nodes of the path being visited by a walk, so that handing a node to a
// callback does not allocate it: a callback is an indirect call, which would make a node
// on the Go stack escape. Callbacks must not keep the node.
type arena struct {
	nodes [maxDepth]Node
	depth int
}

// child returns the node of the key k of n, stored in the arena until pop.
func (a *arena) child(n *Node, k string) *Node {
	c := &a.nodes[a.depth]
	a.depth++
	*c = n.Child(k)
	return c
}

// elemNode returns the node of the element i of n, stored in the arena until pop.
func (a *arena) elemNode(n *Node, i int) *Node {
	c := &a.nodes[a.depth]
	a.depth++
	*c = n.Elem(i)
	return c
}

// root stores n as the first node of the arena.
func (a *arena) root(n Node) *Node {
	c := &a.nodes[a.depth]
	a.depth++
	*c = n
	return c
}

func (a *arena) pop() {
	a.depth--
}

var rootInfo = structinfo.For(reflect.TypeFor[config.Root]())

// Required calls missing for each required field of the configuration that is not set.
func Required(root *config.Root, missing func(n *Node, field string)) {
	e := &ruleWalk{rule: 0, missing: missing}
	n := e.root(RootWithReferences(root))
	e.strct(rootInfo, n, unsafe.Pointer(root))
}

// Enums calls enum for each value of an enum field that is not empty.
func Enums(root *config.Root, enum func(n *Node, s string, valid []string)) {
	e := &ruleWalk{rule: 1, enum: enum}
	n := e.root(Root(root))
	e.strct(rootInfo, n, unsafe.Pointer(root))
}

// zeroIsValid lists the required fields (by struct type) whose zero value is a valid
// value, so it is not reported as missing. Presence comes from ForceSendFields only, and
// these structs have none to record an explicit zero value.
var zeroIsValidType = reflect.TypeFor[jobs.JobParameterDefinition]()

// zeroIsValidKey is the field of zeroIsValidType; an empty default is a common, valid
// job parameter default.
const zeroIsValidKey = "default"

// strct walks the struct t at p, which is nil for a node without fields (a forced nil
// value or a pure reference).
func (e *ruleWalk) strct(t *structinfo.Type, n *Node, p unsafe.Pointer) {
	if e.rule == 0 {
		if p == nil {
			for _, i := range t.Required {
				e.missing(n, t.Fields[i].Key)
			}
			return
		}
		skip := ""
		if t.Typ == zeroIsValidType {
			skip = zeroIsValidKey
		}
		for _, i := range t.Required {
			f := &t.Fields[i]
			if f.Key != skip && !e.present(t, n, p, f) {
				e.missing(n, f.Key)
			}
		}
	} else if p == nil {
		return
	}
	ops := t.Walk[e.rule]
	for i := range ops {
		// Targets and environments are merged into the root.
		if t == rootInfo {
			if k := t.Fields[ops[i].Idx].Key; k == "targets" || k == "environments" {
				continue
			}
		}
		e.field(t, n, p, &ops[i])
	}
}

// present reports whether the required field f of the struct at p counts as set.
// Presence comes from the value and ForceSendFields only, never from locations: a scalar
// or a struct is set if it is not zero or is in ForceSendFields; a pointer, slice, map
// or interface if it is not nil. A value that cannot hold a string can also be replaced
// by a pure reference.
func (e *ruleWalk) present(t *structinfo.Type, n *Node, p unsafe.Pointer, f *structinfo.Field) bool {
	fp := f.Loc.Resolve(p)
	if fp == nil {
		return false
	}
	switch f.Kind {
	case structinfo.KindString:
		return *(*string)(fp) != "" || t.Forced(p, f)
	case structinfo.KindPtr, structinfo.KindSlice, structinfo.KindMap, structinfo.KindInterface:
		return !isNil(f.Kind, fp) || n.IsRef(f.Key)
	case structinfo.KindStruct:
		// A struct whose fields are only references is zero, but set.
		return !isZero(f, fp) || t.Forced(p, f) || n.HasRefs(f.Key)
	default:
		return !isZero(f, fp) || t.Forced(p, f) || n.IsRef(f.Key)
	}
}

func isNil(k structinfo.Kind, p unsafe.Pointer) bool {
	// A slice, map, interface or pointer is nil if its first word is.
	return *(*unsafe.Pointer)(p) == nil
}

func isZero(f *structinfo.Field, p unsafe.Pointer) bool {
	if f.Kind == structinfo.KindBool {
		return !*(*bool)(p)
	}
	return reflect.NewAt(f.Typ, p).Elem().IsZero()
}

type sliceHeader struct {
	data unsafe.Pointer
	len  int
}

// field visits the field of the struct t at p that the walk entry op describes.
func (e *ruleWalk) field(t *structinfo.Type, n *Node, p unsafe.Pointer, op *structinfo.Op) {
	var fp unsafe.Pointer
	if op.Slow {
		fp = t.Fields[op.Idx].Loc.Resolve(p)
		if fp == nil {
			return
		}
	} else {
		fp = unsafe.Add(p, op.Offset)
	}
	ft := op.Elem
	switch {
	case op.Kind == structinfo.KindPtr:
		if q := *(*unsafe.Pointer)(fp); q != nil {
			c := e.child(n, t.Fields[op.Idx].Key)
			e.strct(ft.Elem, c, q)
			e.pop()
		} else if e.rule == 0 && (t.ForcedOp(p, op) || n.IsRef(t.Fields[op.Idx].Key)) {
			// A forced nil value (e.g. "run_job_task:") or one replaced by a pure
			// reference is a node without fields.
			c := e.child(n, t.Fields[op.Idx].Key)
			e.strct(ft.Elem, c, nil)
			e.pop()
		}
	case op.Enum:
		if s := *(*string)(fp); s != "" || t.ForcedOp(p, op) {
			c := e.child(n, t.Fields[op.Idx].Key)
			e.enum(c, s, ft.Enum)
			e.pop()
		}
	case op.Kind == structinfo.KindStruct:
		f := &t.Fields[op.Idx]
		// A zero struct has nothing to report to the enum rule.
		if e.rule == 1 || !isZero(f, fp) || t.ForcedOp(p, op) || n.HasRefs(f.Key) {
			c := e.child(n, f.Key)
			e.strct(ft, c, fp)
			e.pop()
		}
	case op.Kind == structinfo.KindSlice:
		if (*sliceHeader)(fp).len > 0 {
			c := e.child(n, t.Fields[op.Idx].Key)
			e.container(ft, c, fp)
			e.pop()
		}
	case op.Kind == structinfo.KindMap:
		// An empty map has no elements to visit.
		if *(*unsafe.Pointer)(fp) != nil {
			c := e.child(n, t.Fields[op.Idx].Key)
			e.container(ft, c, fp)
			e.pop()
		}
	}
}

// container visits the elements of the slice or map t at p, whose node is c.
func (e *ruleWalk) container(t *structinfo.Type, c *Node, p unsafe.Pointer) {
	if t.Kind == structinfo.KindSlice {
		h := (*sliceHeader)(p)
		for i := range h.len {
			el := e.elemNode(c, i)
			e.elem(t.Elem, el, unsafe.Add(h.data, uintptr(i)*t.Elem.Size))
			e.pop()
		}
		return
	}
	if t.MapKind == structinfo.MapStringPtr {
		for k, q := range *(*map[string]unsafe.Pointer)(p) {
			if k == "" {
				continue
			}
			el := e.child(c, k)
			e.elem(t.Elem, el, unsafe.Pointer(&q))
			e.pop()
		}
		return
	}
	it := reflect.NewAt(t.Typ, p).Elem().MapRange()
	for it.Next() {
		k := it.Key().String()
		if k == "" {
			continue
		}
		el := e.child(c, k)
		v := it.Value()
		if t.Elem.Kind == structinfo.KindPtr {
			q := v.UnsafePointer()
			e.elem(t.Elem, el, unsafe.Pointer(&q))
		} else {
			cp := reflect.New(t.Elem.Typ)
			cp.Elem().Set(v)
			e.elem(t.Elem, el, cp.UnsafePointer())
		}
		e.pop()
	}
}

// elem visits the element of type t at p, whose node is el.
func (e *ruleWalk) elem(t *structinfo.Type, el *Node, p unsafe.Pointer) {
	switch {
	case t.Enum != nil:
		e.enum(el, *(*string)(p), t.Enum)
	case t.Kind == structinfo.KindPtr:
		e.strct(t.Elem, el, *(*unsafe.Pointer)(p))
	case t.Kind == structinfo.KindStruct:
		e.strct(t, el, p)
	case t.Kind == structinfo.KindSlice || t.Kind == structinfo.KindMap:
		e.container(t, el, p)
	}
}

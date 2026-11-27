// Package structinfo holds cached, per-type information about the configuration structs
// (field locations, kinds, required fields, enums) so that walks do not use reflection per
// value: a field is reached by adding an offset to the address of its struct.
package structinfo

import (
	"reflect"
	"slices"
	"sync"
	"unsafe"

	"github.com/databricks/cli/libs/structs/structtag"
	"github.com/databricks/cli/libs/structs/structvar"
)

type Kind uint8

const (
	KindOther Kind = iota
	KindBool
	KindInt
	KindFloat
	KindString
	KindInterface
	KindStruct
	KindPtr
	KindSlice
	KindMap
)

// MapKind classifies a map by the layout of its keys and values.
type MapKind uint8

const (
	// MapOther is any other map: use reflection.
	MapOther MapKind = iota
	// MapStringString is a map with string keys and string values (of any named string
	// types): it can be read as map[string]string.
	MapStringString
	// MapStringPtr is a map with string keys and pointer values: it can be read as
	// map[string]unsafe.Pointer.
	MapStringPtr
)

// Rules that a type can have something to do for, as a bit set.
const (
	RuleRequired uint8 = 1 << iota
	RuleEnum
	// RuleStrings is set for the types that contain strings (or dynamic values that can).
	RuleStrings
	numRules = 3
)

// Step is one hop of a path to a field through a struct embedded by pointer.
type Step struct {
	Offset uintptr
	Deref  bool // the value at Offset is a pointer to the next struct
}

// Loc locates a field in a struct: Offset from the struct's address, or, if a pointer
// embed is on the way, Steps.
type Loc struct {
	Offset uintptr
	Steps  []Step
}

// Resolve returns the address of the field in the struct at base, or nil if a pointer
// embed on the way is nil.
func (l *Loc) Resolve(base unsafe.Pointer) unsafe.Pointer {
	if l.Steps == nil {
		return unsafe.Add(base, l.Offset)
	}
	p := base
	for _, s := range l.Steps {
		p = unsafe.Add(p, s.Offset)
		if s.Deref {
			p = *(*unsafe.Pointer)(p)
			if p == nil {
				return nil
			}
		}
	}
	return p
}

// Op is a field to visit by a walk, in the compact form the walk loop reads.
type Op struct {
	Key    string
	Offset uintptr
	Elem   *Type // the type of the field
	Idx    uint16
	FSF    int16 // index in Type.FSFs, or -1
	// FSFOffset is the offset of the governing ForceSendFields if it is directly in the
	// struct (FSFDirect).
	FSFOffset uintptr
	FSFDirect bool
	Kind      Kind
	Slow      bool // the field is behind a pointer embed: use Fields[Idx].Loc
	Enum      bool // the field's type is an enum
	// NoOmitEmpty is set for a field without omitempty: its zero value is always sent.
	NoOmitEmpty bool
}

// Field is a JSON field of a struct, with embedded structs flattened.
type Field struct {
	Key      string
	GoName   string
	Loc      Loc
	Kind     Kind
	Required bool // no omitempty
	// Hidden is set for internal and read-only fields: the validators skip them and what
	// is below them, other walks visit them.
	Hidden bool
	Typ    reflect.Type
	Type   *Type // the type of the field
	FSF    int   // index in Type.FSFs of the ForceSendFields governing the field, or -1
}

// Type is the cached information of a Go type.
type Type struct {
	Typ    reflect.Type
	Kind   Kind
	Size   uintptr
	Fields []Field
	FSFs   []Loc // ForceSendFields slices, referenced by Field.FSF
	Elem   *Type // pointer target, slice or map element
	Enum   []string
	// MapKind tells how a map can be read without reflection.
	MapKind MapKind

	// Has is the set of rules (Rule* bits) with something to do at or below the type.
	Has uint8
	// Walk[r] lists the fields that need visiting for rule r: the field has something to
	// do itself, or the type below it has. Each entry is small and they are contiguous,
	// so that a walk reads one cache line per few fields.
	Walk [numRules][]Op
	// Required lists the indices of the required Fields.
	Required []uint16
}

var (
	mu    sync.Mutex
	cache = map[reflect.Type]*Type{}
	all   []*Type
)

// For returns the information of typ, building it (and the types below it) on first use.
func For(typ reflect.Type) *Type {
	mu.Lock()
	defer mu.Unlock()
	if t, ok := cache[typ]; ok {
		return t
	}
	n := len(all)
	t := build(typ)
	finish(all[n:])
	return t
}

func kindOf(k reflect.Kind) Kind {
	switch k {
	case reflect.Bool:
		return KindBool
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return KindInt
	case reflect.Float32, reflect.Float64:
		return KindFloat
	case reflect.String:
		return KindString
	case reflect.Interface:
		return KindInterface
	case reflect.Struct:
		return KindStruct
	case reflect.Pointer:
		return KindPtr
	case reflect.Slice:
		return KindSlice
	case reflect.Map:
		return KindMap
	default:
		return KindOther
	}
}

func build(typ reflect.Type) *Type {
	if t, ok := cache[typ]; ok {
		return t
	}
	t := &Type{Typ: typ, Kind: kindOf(typ.Kind()), Size: typ.Size()}
	cache[typ] = t
	all = append(all, t)
	switch typ.Kind() {
	case reflect.Pointer, reflect.Slice, reflect.Map:
		t.Elem = build(typ.Elem())
		if typ.Kind() == reflect.Map && typ.Key().Kind() == reflect.String {
			switch t.Elem.Kind {
			case KindString:
				t.MapKind = MapStringString
			case KindPtr:
				t.MapKind = MapStringPtr
			default:
			}
		}
	case reflect.String:
		t.Enum = enumValues(typ)
	case reflect.Struct:
		buildStruct(t)
	default:
	}
	return t
}

// enumValues returns the values of an enum type: a string type with a Values method
// returning a slice of the type.
func enumValues(typ reflect.Type) []string {
	m, ok := typ.MethodByName("Values")
	if !ok {
		m, ok = reflect.PointerTo(typ).MethodByName("Values")
	}
	if !ok || m.Type.NumIn() != 1 || m.Type.NumOut() != 1 {
		return nil
	}
	out := m.Type.Out(0)
	if out.Kind() != reflect.Slice || out.Elem() != typ {
		return nil
	}
	recv := reflect.New(typ)
	vals := recv.MethodByName("Values").Call(nil)[0]
	var res []string
	for i := range vals.Len() {
		res = append(res, vals.Index(i).String())
	}
	return res
}

// locate returns the location of the field with the index path in typ.
func locate(typ reflect.Type, index []int) Loc {
	var steps []Step
	deref := false
	t := typ
	var off uintptr
	for i, x := range index {
		if i > 0 && t.Kind() == reflect.Pointer {
			steps = append(steps, Step{Offset: off, Deref: true})
			off = 0
			t = t.Elem()
			deref = true
		}
		f := t.Field(x)
		off += f.Offset
		t = f.Type
	}
	if !deref {
		return Loc{Offset: off}
	}
	return Loc{Steps: append(steps, Step{Offset: off})}
}

func buildStruct(t *Type) {
	info := structvar.GetStructInfo(t.Typ)
	for _, g := range info.ForceSendGroups {
		t.FSFs = append(t.FSFs, locate(t.Typ, g))
	}
	for _, fi := range info.List {
		sf := t.Typ.FieldByIndex(fi.Index)
		bt := structtag.BundleTag(sf.Tag.Get("bundle"))
		f := Field{
			Hidden:   bt.Internal() || bt.ReadOnly(),
			Key:      fi.Key,
			GoName:   fi.GolangName,
			Loc:      locate(t.Typ, fi.Index),
			Kind:     kindOf(sf.Type.Kind()),
			Required: fi.ForceEmpty,
			Typ:      sf.Type,
			FSF:      fi.ForceSendGroup,
		}
		f.Type = build(sf.Type)
		t.Fields = append(t.Fields, f)
	}
}

// finish computes the rule sets and walk lists of the new types ts to a fixpoint (types
// can be recursive).
func finish(ts []*Type) {
	own := func(t *Type, f *Field) uint8 {
		var r uint8
		if f.Required {
			r |= RuleRequired
		}
		if f.Type.Enum != nil {
			r |= RuleEnum
		}
		return r
	}
	for changed := true; changed; {
		changed = false
		for _, t := range ts {
			has := t.Has
			switch t.Kind {
			case KindStruct:
				for i := range t.Fields {
					f := &t.Fields[i]
					if f.Hidden {
						has |= f.Type.Has & RuleStrings
						continue
					}
					has |= own(t, f) | f.Type.Has
				}
			case KindPtr, KindSlice, KindMap:
				has |= t.Elem.Has
			case KindString:
				has |= RuleStrings
				if t.Enum != nil {
					has |= RuleEnum
				}
			case KindInterface:
				has |= RuleStrings
			default:
			}
			if t.Kind == KindStruct && structvar.IsSDKNativeType(t.Typ) {
				// Represented as a string in the configuration: nothing to visit inside.
				has &^= RuleStrings
			}
			if has != t.Has {
				t.Has = has
				changed = true
			}
		}
	}
	for _, t := range ts {
		if t.Kind != KindStruct {
			continue
		}
		for i := range t.Fields {
			f := &t.Fields[i]
			if f.Required && !f.Hidden {
				t.Required = append(t.Required, uint16(i))
			}
			for r := range numRules {
				bit := uint8(1) << r
				if f.Hidden && bit != RuleStrings {
					continue
				}
				if f.Type.Has&bit != 0 {
					op := Op{
						Key: f.Key, NoOmitEmpty: f.Required,
						Offset: f.Loc.Offset, Elem: f.Type, Idx: uint16(i), FSF: int16(f.FSF),
						Kind: f.Kind, Slow: f.Loc.Steps != nil, Enum: f.Type.Enum != nil,
					}
					if f.FSF >= 0 && t.FSFs[f.FSF].Steps == nil {
						op.FSFOffset, op.FSFDirect = t.FSFs[f.FSF].Offset, true
					}
					t.Walk[r] = append(t.Walk[r], op)
				}
			}
		}
	}
}

// ForcedOp is [Type.Forced] for a walk entry.
func (t *Type) ForcedOp(p unsafe.Pointer, op *Op) bool {
	if op.FSF < 0 {
		return false
	}
	if op.FSFDirect && len(*(*[]string)(unsafe.Add(p, op.FSFOffset))) == 0 {
		return false
	}
	return t.Forced(p, &t.Fields[op.Idx])
}

// Forced reports whether the Go field name is listed in the ForceSendFields governing f
// in the struct at p.
func (t *Type) Forced(p unsafe.Pointer, f *Field) bool {
	if f.FSF < 0 {
		return false
	}
	fp := t.FSFs[f.FSF].Resolve(p)
	if fp == nil {
		return false
	}
	fsf := *(*[]string)(fp)
	if len(fsf) == 0 {
		return false
	}
	return slices.Contains(fsf, f.GoName)
}

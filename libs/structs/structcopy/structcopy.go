// Package structcopy compiles a reflection-based copier between two struct types.
// The copier moves fields that match by JSON name, filters ForceSendFields to the
// destination type, and applies only lossless conversions. The plan is compiled once
// from the two types (see Compile); executing it on any values of those types cannot fail.
//
// It underpins the direct engine's automatic RemapState (bundle/direct/dresources): when a
// resource's state type is a plain subset of its remote type, no hand-written copy is needed.
package structcopy

import (
	"fmt"
	"reflect"

	"github.com/databricks/cli/libs/structs/structaccess"
	"github.com/databricks/cli/libs/structs/structtag"
	"github.com/databricks/cli/libs/utils"
)

// Copier copies fields from a source struct into a fresh destination struct by matching
// JSON field names. Build one with Compile; the compiled plan cannot fail at Copy time.
type Copier struct {
	dstElem reflect.Type // struct type behind the destination pointer type
	ops     []copyOp
}

type copyOp struct {
	// forceSendFields ops recompute a ForceSendFields slice; field-copy ops move one field.
	forceSendFields bool

	dstIndex []int // FieldByIndex path in the destination struct

	// field-copy ops:
	srcIndex []int        // FieldByIndex path in the source struct
	convert  bool         // use reflect.Convert (distinct but same-underlying types) instead of a direct set
	dstType  reflect.Type // target type for Convert

	// forceSendFields ops:
	ownerType         reflect.Type // struct type owning the destination ForceSendFields slice, for validity filtering
	srcForceSendIndex []int        // FieldByIndex path to the matching source ForceSendFields slice, nil if the source has none
}

// jsonFieldInfo locates a top-level JSON field within a struct, resolved through
// encoding/json-flattened embeds.
type jsonFieldInfo struct {
	index []int
	typ   reflect.Type
}

// forceSendTarget locates a ForceSendFields slice within a struct tree and the
// struct type that owns it.
type forceSendTarget struct {
	index     []int
	ownerType reflect.Type
}

// Compile builds the copy plan for srcType -> dstType (both pointers to structs). It returns
// an error if a destination field that also exists on the source cannot be copied without a
// value-changing conversion, or if either type has a shape the copier does not support
// (an anonymous pointer embed, or a JSON-name collision), so callers reject such a type pair
// up front rather than silently dropping a field or panicking at copy time.
func Compile(srcType, dstType reflect.Type) (*Copier, error) {
	dstElem := dstType.Elem()
	srcElem := srcType.Elem()

	dstFields, dstForceSend, err := flattenStruct(dstElem, nil)
	if err != nil {
		return nil, fmt.Errorf("destination %s: %w", dstElem, err)
	}
	srcFields, srcForceSend, err := flattenStruct(srcElem, nil)
	if err != nil {
		return nil, fmt.Errorf("source %s: %w", srcElem, err)
	}

	var ops []copyOp
	for name, dst := range dstFields {
		src, ok := srcFields[name]
		if !ok {
			// Field absent from the source type is left zero in the destination.
			continue
		}
		switch {
		case dst.typ == src.typ:
			ops = append(ops, copyOp{dstIndex: dst.index, srcIndex: src.index, dstType: dst.typ})
		case safeConvert(dst.typ, src.typ):
			ops = append(ops, copyOp{dstIndex: dst.index, srcIndex: src.index, convert: true, dstType: dst.typ})
		default:
			return nil, fmt.Errorf("field %q: destination type %s is not assignable or safely convertible from source type %s", name, dst.typ, src.typ)
		}
	}

	// Match each destination ForceSendFields slice to the source slice that governs the same
	// fields, so a slice is never populated from an unrelated struct's names. The top-level
	// slice (index length 1) pairs with the source's top-level slice; a slice owned by an
	// embedded struct pairs with the source slice owned by that same struct type. An
	// unmatched destination slice is left empty.
	var srcRootForceSend []int
	srcForceSendByOwner := make(map[reflect.Type][]int, len(srcForceSend))
	for _, s := range srcForceSend {
		if len(s.index) == 1 {
			srcRootForceSend = s.index
		}
		srcForceSendByOwner[s.ownerType] = s.index
	}
	for _, target := range dstForceSend {
		srcIdx := srcForceSendByOwner[target.ownerType]
		if len(target.index) == 1 {
			srcIdx = srcRootForceSend
		}
		ops = append(ops, copyOp{forceSendFields: true, dstIndex: target.index, ownerType: target.ownerType, srcForceSendIndex: srcIdx})
	}

	return &Copier{dstElem: dstElem, ops: ops}, nil
}

// Copy builds a fresh destination value (a pointer to the destination struct) populated
// from src (a pointer to the source struct).
func (c *Copier) Copy(src any) any {
	srcVal := reflect.ValueOf(src).Elem()
	dstPtr := reflect.New(c.dstElem)
	dstVal := dstPtr.Elem()

	for _, op := range c.ops {
		if op.forceSendFields {
			var srcForceSend []string
			if op.srcForceSendIndex != nil {
				srcForceSend, _ = reflect.TypeAssert[[]string](srcVal.FieldByIndex(op.srcForceSendIndex))
			}
			filtered := utils.FilterFieldsType(op.ownerType, srcForceSend)
			dstVal.FieldByIndex(op.dstIndex).Set(reflect.ValueOf(filtered))
			continue
		}
		val := srcVal.FieldByIndex(op.srcIndex)
		if op.convert {
			val = val.Convert(op.dstType)
		}
		dstVal.FieldByIndex(op.dstIndex).Set(val)
	}

	return dstPtr.Interface()
}

// safeConvert reports whether a value of src can be converted to dst without changing
// the value or its JSON meaning. Distinct named scalar types with the same underlying
// type (e.g. two string enums) qualify. Kind-changing conversions (int<->string,
// float->int, []byte<->string) do not, because reflect.Convert would silently corrupt
// the value. Composite kinds (struct/map/slice/array) are also excluded: Go permits
// converting between two structs with identical fields even when their JSON tags differ,
// which would silently drop or rename fields on marshal; such fields must match exactly
// (handled by the dst.typ == src.typ branch) or the resource keeps a custom RemapState.
func safeConvert(dst, src reflect.Type) bool {
	return isScalarKind(src.Kind()) && src.ConvertibleTo(dst) && src.Kind() == dst.Kind()
}

// isScalarKind reports whether k is a non-composite kind (bool, numeric, or string),
// for which a same-kind reflect.Convert is value- and JSON-preserving.
func isScalarKind(k reflect.Kind) bool {
	switch k {
	case reflect.Bool, reflect.String,
		reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Float32, reflect.Float64:
		return true
	default:
		return false
	}
}

// flattenStruct enumerates a struct's top-level JSON fields (descending through
// encoding/json-flattened embeds and accumulating the field-index prefix) plus every
// ForceSendFields slice reachable through those embeds. Names follow the same JSON tag
// resolution the diff engine uses, so copied values compare consistently. It returns an
// error for shapes the copier does not model: an anonymous pointer embed (FieldByIndex
// would panic on a nil pointer at copy time) or two fields resolving to the same JSON name
// (encoding/json treats that as ambiguous; the copier would otherwise pick one silently).
func flattenStruct(t reflect.Type, prefix []int) (map[string]jsonFieldInfo, []forceSendTarget, error) {
	fields := make(map[string]jsonFieldInfo)
	var forceSend []forceSendTarget

	add := func(name string, info jsonFieldInfo) error {
		if _, dup := fields[name]; dup {
			return fmt.Errorf("json name %q is declared by more than one field", name)
		}
		fields[name] = info
		return nil
	}

	for i := range t.NumField() {
		sf := t.Field(i)
		if sf.PkgPath != "" {
			continue // unexported
		}
		index := append(append([]int{}, prefix...), i)

		if sf.Name == "ForceSendFields" {
			forceSend = append(forceSend, forceSendTarget{index: index, ownerType: t})
			continue
		}
		if structaccess.IsFlattenedEmbed(sf) {
			if sf.Type.Kind() == reflect.Pointer {
				return nil, nil, fmt.Errorf("anonymous pointer embed %s is not supported", sf.Type)
			}
			nested, nestedForceSend, err := flattenStruct(sf.Type, index)
			if err != nil {
				return nil, nil, err
			}
			for name, info := range nested {
				if err := add(name, info); err != nil {
					return nil, nil, err
				}
			}
			forceSend = append(forceSend, nestedForceSend...)
			continue
		}
		if structaccess.IsSkippedField(sf) || sf.Name == structaccess.EmbeddedSliceFieldName {
			continue
		}

		name := structtag.JSONTag(sf.Tag.Get("json")).Name()
		if name == "" {
			name = sf.Name
		}
		if err := add(name, jsonFieldInfo{index: index, typ: sf.Type}); err != nil {
			return nil, nil, err
		}
	}

	return fields, forceSend, nil
}

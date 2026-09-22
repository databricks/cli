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
	"maps"
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
	ownerType reflect.Type // struct type owning the ForceSendFields slice, for validity filtering
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
// value-changing conversion, so callers can reject such a type pair up front rather than
// silently dropping the field.
func Compile(srcType, dstType reflect.Type) (*Copier, error) {
	dstElem := dstType.Elem()
	srcElem := srcType.Elem()

	dstFields, dstForceSend := flattenStruct(dstElem, nil)
	srcFields, _ := flattenStruct(srcElem, nil)

	var ops []copyOp
	for name, dst := range dstFields {
		src, ok := srcFields[name]
		if !ok {
			// Field absent from the source type is left zero in the destination.
			continue
		}
		switch {
		case dst.typ == src.typ:
			ops = append(ops, copyOp{forceSendFields: false, dstIndex: dst.index, srcIndex: src.index, convert: false, dstType: dst.typ, ownerType: nil})
		case safeConvert(dst.typ, src.typ):
			ops = append(ops, copyOp{forceSendFields: false, dstIndex: dst.index, srcIndex: src.index, convert: true, dstType: dst.typ, ownerType: nil})
		default:
			return nil, fmt.Errorf("field %q: destination type %s is not assignable or safely convertible from source type %s", name, dst.typ, src.typ)
		}
	}

	for _, target := range dstForceSend {
		ops = append(ops, copyOp{forceSendFields: true, dstIndex: target.index, srcIndex: nil, convert: false, dstType: nil, ownerType: target.ownerType})
	}

	return &Copier{dstElem: dstElem, ops: ops}, nil
}

// Copy builds a fresh destination value (a pointer to the destination struct) populated
// from src (a pointer to the source struct).
func (c *Copier) Copy(src any) any {
	srcVal := reflect.ValueOf(src).Elem()
	dstPtr := reflect.New(c.dstElem)
	dstVal := dstPtr.Elem()

	var srcForceSend []string
	for _, op := range c.ops {
		if op.forceSendFields {
			if srcForceSend == nil {
				srcForceSend = rootForceSendFields(srcVal)
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
// the value. Distinct named types with the same underlying type (e.g. two string enums)
// qualify; kind-changing conversions (int<->string, float->int, []byte<->string) do not,
// because reflect.Convert would silently corrupt the value.
func safeConvert(dst, src reflect.Type) bool {
	return src.ConvertibleTo(dst) && src.Kind() == dst.Kind()
}

// rootForceSendFields returns the ForceSendFields slice at the struct's root, or nil.
func rootForceSendFields(v reflect.Value) []string {
	f := v.FieldByName("ForceSendFields")
	if !f.IsValid() || f.Kind() != reflect.Slice {
		return nil
	}
	fields, _ := reflect.TypeAssert[[]string](f)
	return fields
}

// flattenStruct enumerates a struct's top-level JSON fields (descending through
// encoding/json-flattened embeds and accumulating the field-index prefix) plus every
// ForceSendFields slice reachable through those embeds. Names follow the same JSON tag
// resolution the diff engine uses, so copied values compare consistently.
func flattenStruct(t reflect.Type, prefix []int) (map[string]jsonFieldInfo, []forceSendTarget) {
	fields := make(map[string]jsonFieldInfo)
	var forceSend []forceSendTarget

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
			nested, nestedForceSend := flattenStruct(sf.Type, index)
			maps.Copy(fields, nested)
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
		fields[name] = jsonFieldInfo{index: index, typ: sf.Type}
	}

	return fields, forceSend
}

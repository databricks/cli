package dresources

import (
	"fmt"
	"reflect"

	"github.com/databricks/cli/libs/structs/structaccess"
	"github.com/databricks/cli/libs/structs/structtag"
	"github.com/databricks/cli/libs/utils"
)

// copier is a reflection-built RemapState: it copies fields from a remote
// struct into a fresh state struct by matching JSON field names. It replaces the
// hand-written "dumb copy" RemapState methods (see README.md) for resources whose
// state type is a plain subset of the remote type.
//
// The plan is compiled once from the two types (see compileCopier); executing it
// on any pair of values of those types cannot fail. Anything the plan cannot copy
// safely is rejected at compile time, so a resource that needs real logic surfaces
// as a build/init error rather than a silent drift bug — see buildCopiers.
type copier struct {
	stateElem reflect.Type // struct type behind *StateType
	ops       []copyOp
}

type copyOp struct {
	// forceSendFields ops recompute a ForceSendFields slice; field-copy ops move one field.
	forceSendFields bool

	dstIndex []int // FieldByIndex path in the state struct

	// field-copy ops:
	srcIndex []int        // FieldByIndex path in the remote struct
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

// compileCopier builds the copy plan for remoteType -> stateType (both pointers to
// structs). It returns an error if any state field that also exists on the remote cannot
// be copied without a value-changing conversion; such a field needs a custom RemapState.
func compileCopier(remoteType, stateType reflect.Type) (*copier, error) {
	stateElem := stateType.Elem()
	remoteElem := remoteType.Elem()

	dstFields, dstForceSend := flattenStruct(stateElem, nil)
	srcFields, _ := flattenStruct(remoteElem, nil)

	var ops []copyOp
	for name, dst := range dstFields {
		src, ok := srcFields[name]
		if !ok {
			// Field absent from the remote type: always nil/zero in the remapped
			// state. This is the missing_in_remote invariant the planner relies on.
			continue
		}
		switch {
		case dst.typ == src.typ:
			ops = append(ops, copyOp{dstIndex: dst.index, srcIndex: src.index, dstType: dst.typ})
		case safeConvert(dst.typ, src.typ):
			ops = append(ops, copyOp{dstIndex: dst.index, srcIndex: src.index, convert: true, dstType: dst.typ})
		default:
			return nil, fmt.Errorf("field %q: state type %s cannot be copied from remote type %s; implement RemapState", name, dst.typ, src.typ)
		}
	}

	for _, target := range dstForceSend {
		ops = append(ops, copyOp{forceSendFields: true, dstIndex: target.index, ownerType: target.ownerType})
	}

	return &copier{stateElem: stateElem, ops: ops}, nil
}

// safeConvert reports whether a value of src can be converted to dst without changing
// the value. Distinct named types with the same underlying type (e.g. two string enums)
// qualify; kind-changing conversions (int<->string, float->int, []byte<->string) do not,
// because reflect.Convert would silently corrupt the value.
func safeConvert(dst, src reflect.Type) bool {
	return src.ConvertibleTo(dst) && src.Kind() == dst.Kind()
}

// rootForceSendFields returns the ForceSendFields slice of the remote struct's root, or nil.
func rootForceSendFields(remote reflect.Value) []string {
	f := remote.FieldByName("ForceSendFields")
	if !f.IsValid() || f.Kind() != reflect.Slice {
		return nil
	}
	fields, _ := f.Interface().([]string)
	return fields
}

// copy builds a fresh *StateType populated from remote (a *RemoteType).
func (c *copier) copy(remote any) any {
	src := reflect.ValueOf(remote).Elem()
	dstPtr := reflect.New(c.stateElem)
	dst := dstPtr.Elem()

	var srcForceSend []string
	for _, op := range c.ops {
		if op.forceSendFields {
			if srcForceSend == nil {
				srcForceSend = rootForceSendFields(src)
			}
			filtered := utils.FilterFieldsType(op.ownerType, srcForceSend)
			dst.FieldByIndex(op.dstIndex).Set(reflect.ValueOf(filtered))
			continue
		}
		val := src.FieldByIndex(op.srcIndex)
		if op.convert {
			val = val.Convert(op.dstType)
		}
		dst.FieldByIndex(op.dstIndex).Set(val)
	}

	return dstPtr.Interface()
}

// flattenStruct enumerates a struct's top-level JSON fields (descending through
// encoding/json-flattened embeds and accumulating the field-index prefix) plus every
// ForceSendFields slice reachable through those embeds. Names follow the same JSON tag
// resolution the diff engine uses, so copied state compares consistently.
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
			for name, info := range nested {
				fields[name] = info
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
		fields[name] = jsonFieldInfo{index: index, typ: sf.Type}
	}

	return fields, forceSend
}

package structaccess

import (
	"reflect"
	"sync"

	"github.com/databricks/cli/libs/structs/structtag"
)

const forceSendFieldsName = "ForceSendFields"

// Field describes one struct field the way encoding/json and the structs packages see it.
type Field struct {
	// Index is the field's index for reflect.Value.Field.
	Index int

	StructField reflect.StructField

	// Name is the JSON key: the json tag name, or the Go field name when the tag has none.
	Name string

	OmitEmpty bool

	// Flattened marks an anonymous struct that encoding/json promotes into the parent, so its
	// fields appear at the parent's path. An anonymous field carrying a json name is not one of
	// these: encoding/json serializes it as a nested object, so it is a named field.
	Flattened bool

	// EmbeddedSlice marks the EmbeddedSliceFieldName field, whose elements appear at the parent's path.
	EmbeddedSlice bool
}

// StructInfo is the per-type view of a struct's fields, computed once per type.
type StructInfo struct {
	// Fields lists the exported fields in declaration order, except ForceSendFields and
	// fields tagged exactly `json:"-"`.
	Fields []Field

	// ForceSendFieldsIndex is the index of the ForceSendFields []string the struct declares
	// itself, or -1. A ForceSendFields promoted from an embedded struct only governs that
	// struct's fields: the SDK marshaller rejects names of fields its struct does not have.
	ForceSendFieldsIndex int
}

// Keyed by reflect.Type; struct walkers visit the same types many times.
var structInfoCache sync.Map

// GetStructInfo returns the StructInfo of struct type t.
func GetStructInfo(t reflect.Type) *StructInfo {
	if cached, ok := structInfoCache.Load(t); ok {
		return cached.(*StructInfo)
	}
	info := newStructInfo(t)
	structInfoCache.Store(t, info)
	return info
}

func newStructInfo(t reflect.Type) *StructInfo {
	info := &StructInfo{ForceSendFieldsIndex: -1}
	for i := range t.NumField() {
		sf := t.Field(i)
		if sf.Name == forceSendFieldsName {
			if !sf.Anonymous && sf.Type == reflect.TypeFor[[]string]() {
				info.ForceSendFieldsIndex = i
			}
			continue
		}
		if !sf.IsExported() {
			continue
		}
		if IsFlattenedEmbed(sf) {
			info.Fields = append(info.Fields, Field{Index: i, StructField: sf, Flattened: true})
			continue
		}
		if IsSkippedField(sf) {
			continue
		}

		jsonTag := structtag.JSONTag(sf.Tag.Get("json"))
		name := jsonTag.Name()
		if name == "" {
			name = sf.Name
		}
		info.Fields = append(info.Fields, Field{
			Index:         i,
			StructField:   sf,
			Name:          name,
			OmitEmpty:     jsonTag.OmitEmpty(),
			EmbeddedSlice: sf.Name == EmbeddedSliceFieldName,
		})
	}
	return info
}

// ForceSendFields returns the ForceSendFields that struct v declares itself, or nil.
func (info *StructInfo) ForceSendFields(v reflect.Value) []string {
	if info.ForceSendFieldsIndex < 0 {
		return nil
	}
	result, _ := reflect.TypeAssert[[]string](v.Field(info.ForceSendFieldsIndex))
	return result
}

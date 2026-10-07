package structvar

import (
	"reflect"
	"slices"
	"sync"

	"github.com/databricks/cli/libs/structs/structtag"
	sdkduration "github.com/databricks/databricks-sdk-go/common/types/duration"
	sdkfieldmask "github.com/databricks/databricks-sdk-go/common/types/fieldmask"
	sdktime "github.com/databricks/databricks-sdk-go/common/types/time"
)

// StructInfo holds the type information of a struct for converting between
// the configuration tree and the typed value: JSON names, omitempty, ForceSendFields.
type StructInfo struct {
	// FieldNames is ordered list of fields
	FieldNames []string

	// Fields maps the JSON-name of the field to the field's index for use with [FieldByIndex].
	Fields map[string][]int

	// Tracks which fields do not have omitempty annotation
	ForceEmpty map[string]bool

	// Maps JSON-name of the field to Golang struct name
	GolangNames map[string]string

	// ForceSendFieldsIndex maps the JSON-name of the field to the index path (for
	// use with [reflect.Value.FieldByIndex]) of the ForceSendFields slice that
	// governs it: the one declared by the struct that also declares the field.
	// The path is static per type, so we resolve it once here rather than walking
	// the value at conversion time. It can be more than one element deep because a
	// field's declaring struct may be embedded several levels down (e.g.
	// PostgresProject -> PostgresProjectConfig -> ProjectSpec). A field whose
	// declaring struct has no ForceSendFields has no entry.
	ForceSendFieldsIndex map[string][]int
}

// StructInfoCache caches type information.
var StructInfoCache = make(map[reflect.Type]StructInfo)

// StructInfoCacheLock guards concurrent access to StructInfoCache.
var StructInfoCacheLock sync.Mutex

// getStructInfo returns the [StructInfo] for the given type.
// It lazily populates a cache, so the first call for a given
// type is slower than subsequent calls for that same type.
func GetStructInfo(typ reflect.Type) StructInfo {
	StructInfoCacheLock.Lock()
	defer StructInfoCacheLock.Unlock()

	si, ok := StructInfoCache[typ]
	if !ok {
		si = buildStructInfo(typ)
		StructInfoCache[typ] = si
	}

	return si
}

// buildStructInfo populates a new [StructInfo] for the given type.
func buildStructInfo(typ reflect.Type) StructInfo {
	out := StructInfo{
		Fields:               make(map[string][]int),
		ForceEmpty:           make(map[string]bool),
		GolangNames:          make(map[string]string),
		ForceSendFieldsIndex: make(map[string][]int),
	}

	// Queue holds the indexes of the structs to visit.
	// It is initialized with a single empty slice to visit the top level struct.
	queue := [][]int{{}}
	for i := 0; i < len(queue); i++ {
		prefix := queue[i]

		// Traverse embedded anonymous types (if prefix is non-empty).
		styp := typ
		if len(prefix) > 0 {
			styp = styp.FieldByIndex(prefix).Type
		}

		// Dereference pointer type.
		if styp.Kind() == reflect.Pointer {
			styp = styp.Elem()
		}

		// Index path to the ForceSendFields declared by this struct, if any. All
		// fields declared directly by this struct are governed by it. The len==1
		// check excludes a ForceSendFields promoted from an embedded struct: that
		// one governs the embedded struct's own fields, which we visit separately.
		var forceSendFieldsIndex []int
		if sf, ok := styp.FieldByName("ForceSendFields"); ok && len(sf.Index) == 1 {
			forceSendFieldsIndex = append(slices.Clone(prefix), sf.Index...)
		}

		nf := styp.NumField()
		for j := range nf {
			sf := styp.Field(j)

			// Recurse into anonymous fields.
			if sf.Anonymous {
				queue = append(queue, append(prefix, sf.Index...))
				continue
			}

			jtag := structtag.JSONTag(sf.Tag.Get("json"))
			name := jtag.Name()
			if name == "" || name == "-" {
				continue
			}

			// Top level fields always take precedence.
			// Therefore, if it is already set, we ignore it.
			if _, ok := out.Fields[name]; ok {
				continue
			}

			out.FieldNames = append(out.FieldNames, name)
			out.Fields[name] = append(prefix, sf.Index...)
			if !jtag.OmitEmpty() && !jtag.OmitZero() {
				out.ForceEmpty[name] = true
			}
			out.GolangNames[name] = sf.Name

			// The field is declared directly in this struct, so it is governed by
			// this struct's ForceSendFields (if it has one).
			if forceSendFieldsIndex != nil {
				out.ForceSendFieldsIndex[name] = forceSendFieldsIndex
			}
		}
	}

	return out
}

type FieldValue struct {
	Key      string
	Value    reflect.Value
	IsForced bool
}

func (s *StructInfo) FieldValues(v reflect.Value) []FieldValue {
	out := make([]FieldValue, 0, len(s.Fields))

	for _, k := range s.FieldNames {
		fv := FieldByIndex(v, s.Fields[k])

		if fv.IsValid() {
			isForced := true

			// TODO: we should use isEmptyForOmitEmpty instead of IsZero()
			if fv.IsZero() {
				isForced = s.IsForceSend(v, k)
			}

			out = append(out, FieldValue{
				Key:      k,
				Value:    fv,
				IsForced: isForced,
			})
		}
	}

	return out
}

// isForceSend reports whether the field named k is listed in the ForceSendFields
// that governs it (see StructInfo.ForceSendFieldsIndex).
func (s *StructInfo) IsForceSend(v reflect.Value, k string) bool {
	index, ok := s.ForceSendFieldsIndex[k]
	if !ok {
		return false
	}
	fsf := FieldByIndex(v, index)
	if !fsf.IsValid() {
		return false
	}
	return slices.Contains(fsf.Interface().([]string), s.GolangNames[k])
}

// fieldByIndex resolves the value at the given index path, dereferencing embedded
// pointer structs on the way. It returns an invalid value if a nil pointer is met.
func FieldByIndex(v reflect.Value, index []int) reflect.Value {
	for i, x := range index {
		if i > 0 {
			if v.Kind() == reflect.Pointer && v.Type().Elem().Kind() == reflect.Struct {
				if v.IsNil() {
					return reflect.Value{}
				}
				v = v.Elem()
			}
		}
		v = v.Field(x)
	}
	return v
}

// GetOrNewFieldByIndex resolves the value at the given index path within an addressable
// struct, allocating intermediate structs embedded as pointer types along the way.
func GetOrNewFieldByIndex(v reflect.Value, index []int) reflect.Value {
	for i, x := range index {
		if i > 0 {
			if v.Kind() == reflect.Pointer {
				if v.IsNil() {
					v.Set(reflect.New(v.Type().Elem()))
				}
				v = v.Elem()
			}
		}
		v = v.Field(x)
	}
	return v
}

// sdkNativeTypes are SDK types with custom JSON marshaling to and from a string.
// The configuration represents them as strings.
var sdkNativeTypes = []reflect.Type{
	reflect.TypeFor[sdkduration.Duration](),   // Protobuf duration format (e.g., "300s")
	reflect.TypeFor[sdktime.Time](),           // RFC3339 timestamp format (e.g., "2023-12-25T10:30:00Z")
	reflect.TypeFor[sdkfieldmask.FieldMask](), // Comma-separated paths (e.g., "name,age,email")
}

// IsSDKNativeType reports whether typ is an SDK type represented as a string.
func IsSDKNativeType(typ reflect.Type) bool {
	return slices.Contains(sdkNativeTypes, typ)
}

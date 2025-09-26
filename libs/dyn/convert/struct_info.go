package convert

import (
	"reflect"
	"slices"
	"sync"

	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/structs/structtag"
)

// structInfo holds the type information we need to efficiently
// convert data from a [dyn.Value] to a Go struct.
type structInfo struct {
	// FieldNames is ordered list of fields
	FieldNames []string

	// Fields maps the JSON-name of the field to the field's index for use with [FieldByIndex].
	Fields map[string][]int

	// ValueField maps to the field with a [dyn.Value].
	// The underlying type is expected to only have one of these.
	ValueField []int

	// Tracks which fields do not have omitempty annotation
	ForceEmpty map[string]bool

	// Maps JSON-name of the field to Golang struct name
	GolangNames map[string]string

	// ForceSendFieldsStructKey maps the JSON-name of the field to which ForceSendFields slice it belongs to:
	// -1 for direct fields, embedded struct index for embedded fields
	ForceSendFieldsStructKey map[string]int
}

// structInfoCache caches type information.
var structInfoCache = make(map[reflect.Type]structInfo)

// structInfoCacheLock guards concurrent access to structInfoCache.
var structInfoCacheLock sync.Mutex

// getStructInfo returns the [structInfo] for the given type.
// It lazily populates a cache, so the first call for a given
// type is slower than subsequent calls for that same type.
func getStructInfo(typ reflect.Type) structInfo {
	structInfoCacheLock.Lock()
	defer structInfoCacheLock.Unlock()

	si, ok := structInfoCache[typ]
	if !ok {
		si = buildStructInfo(typ)
		structInfoCache[typ] = si
	}

	return si
}

// buildStructInfo populates a new [structInfo] for the given type.
func buildStructInfo(typ reflect.Type) structInfo {
	out := structInfo{
		Fields:                   make(map[string][]int),
		ForceEmpty:               make(map[string]bool),
		GolangNames:              make(map[string]string),
		ForceSendFieldsStructKey: make(map[string]int),
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

		nf := styp.NumField()
		for j := range nf {
			sf := styp.Field(j)

			// Recurse into anonymous fields.
			if sf.Anonymous {
				queue = append(queue, append(prefix, sf.Index...))
				continue
			}

			// If this field has type [dyn.Value], we populate it with the source [dyn.Value] from [ToTyped].
			if sf.IsExported() && sf.Type == configValueType {
				if out.ValueField != nil {
					panic("multiple dyn.Value fields")
				}
				out.ValueField = append(prefix, sf.Index...)
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

			// Determine which ForceSendFields this field belongs to
			if len(prefix) == 0 {
				// Direct field on the main struct
				out.ForceSendFieldsStructKey[name] = -1
			} else {
				// Field on embedded struct
				out.ForceSendFieldsStructKey[name] = prefix[0]
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

func (s *structInfo) FieldValues(v reflect.Value) []FieldValue {
	out := make([]FieldValue, 0, len(s.Fields))

	// Collect ForceSendFields from all levels for field inclusion logic
	forceSendFieldsMap := getForceSendFieldsForFromTyped(v)

	for _, k := range s.FieldNames {
		index := s.Fields[k]
		fv := v

		// Locate value in struct (it could be an embedded type).
		for i, x := range index {
			if i > 0 {
				if fv.Kind() == reflect.Pointer && fv.Type().Elem().Kind() == reflect.Struct {
					if fv.IsNil() {
						fv = reflect.Value{}
						break
					}
					fv = fv.Elem()
				}
			}
			fv = fv.Field(x)
		}

		if fv.IsValid() {
			isForced := true

			// TODO: we should use isEmptyForOmitEmpty instead of IsZero()
			if fv.IsZero() {
				goName := s.GolangNames[k]
				structKey := s.ForceSendFieldsStructKey[k]
				forceSendFields := forceSendFieldsMap[structKey]
				isForced = slices.Contains(forceSendFields, goName)
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

// Type of [dyn.Value].
var configValueType = reflect.TypeOf((*dyn.Value)(nil)).Elem()

// getForceSendFieldsValues collects ForceSendFields reflect.Values
// Returns map[structKey]reflect.Value where structKey is -1 for direct fields, embedded index for embedded fields
func getForceSendFieldsValues(v reflect.Value) map[int]reflect.Value {
	if !v.IsValid() || v.Type().Kind() != reflect.Struct {
		return make(map[int]reflect.Value)
	}

	result := make(map[int]reflect.Value)

	for i := range v.Type().NumField() {
		field := v.Type().Field(i)
		fieldValue := v.Field(i)

		if field.Name == "ForceSendFields" && !field.Anonymous {
			// Direct ForceSendFields (structKey = -1)
			result[-1] = fieldValue
		} else if field.Anonymous {
			// Embedded struct - check for ForceSendFields inside it
			if embeddedStruct := getEmbeddedStruct(fieldValue); embeddedStruct.IsValid() {
				if forceSendField := embeddedStruct.FieldByName("ForceSendFields"); forceSendField.IsValid() {
					result[i] = forceSendField
				}
			}
		}
	}

	return result
}

// getForceSendFieldsForFromTyped collects ForceSendFields values for FromTyped operations
// Returns map[structKey][]fieldName where structKey is -1 for direct fields, embedded index for embedded fields
func getForceSendFieldsForFromTyped(v reflect.Value) map[int][]string {
	values := getForceSendFieldsValues(v)
	result := make(map[int][]string)

	for structKey, fieldValue := range values {
		if fields, ok := fieldValue.Interface().([]string); ok {
			result[structKey] = fields
		}
	}

	return result
}

// getEmbeddedStruct handles embedded struct access - never creates nil pointers
func getEmbeddedStruct(fieldValue reflect.Value) reflect.Value {
	if fieldValue.Kind() == reflect.Pointer {
		if fieldValue.IsNil() {
			return reflect.Value{} // Don't create, just return invalid
		}
		fieldValue = fieldValue.Elem()
	}
	if fieldValue.Kind() == reflect.Struct {
		return fieldValue
	}
	return reflect.Value{}
}

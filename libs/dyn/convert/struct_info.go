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

func (s *structInfo) FieldValues(v reflect.Value) []FieldValue {
	out := make([]FieldValue, 0, len(s.Fields))

	for _, k := range s.FieldNames {
		fv := fieldByIndex(v, s.Fields[k])

		if fv.IsValid() {
			isForced := true

			// TODO: we should use isEmptyForOmitEmpty instead of IsZero()
			if fv.IsZero() {
				isForced = s.isForceSend(v, k)
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
// that governs it (see structInfo.ForceSendFieldsIndex).
func (s *structInfo) isForceSend(v reflect.Value, k string) bool {
	index, ok := s.ForceSendFieldsIndex[k]
	if !ok {
		return false
	}
	fsf := fieldByIndex(v, index)
	if !fsf.IsValid() {
		return false
	}
	return slices.Contains(fsf.Interface().([]string), s.GolangNames[k])
}

// fieldByIndex resolves the value at the given index path, dereferencing embedded
// pointer structs on the way. It returns an invalid value if a nil pointer is met.
func fieldByIndex(v reflect.Value, index []int) reflect.Value {
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

// Type of [dyn.Value].
var configValueType = reflect.TypeFor[dyn.Value]()

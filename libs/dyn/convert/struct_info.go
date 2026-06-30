package convert

import (
	"reflect"
	"slices"
	"strconv"
	"strings"
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

	// ForceSendFieldsStructKey maps the JSON-name of the field to the struct that
	// owns the ForceSendFields slice it belongs to. The value is the index path to
	// that struct (see indexPathKey); "" is the top-level struct. The path can be
	// more than one element deep because a field's declaring struct may be embedded
	// several levels down (e.g. PostgresProject -> PostgresProjectConfig -> ProjectSpec).
	ForceSendFieldsStructKey map[string]string
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
		ForceSendFieldsStructKey: make(map[string]string),
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

			// The field is declared directly in the struct reached by prefix, so
			// its ForceSendFields lives there. prefix is empty for the top-level struct.
			out.ForceSendFieldsStructKey[name] = indexPathKey(prefix)
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
	forceSendFieldsMap := getForceSendFieldsValues(v)

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
				if fieldValue, exists := forceSendFieldsMap[structKey]; exists {
					forceSendFields := fieldValue.Interface().([]string)
					isForced = slices.Contains(forceSendFields, goName)
				} else {
					isForced = false
				}
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
var configValueType = reflect.TypeFor[dyn.Value]()

// getForceSendFieldsValues collects the ForceSendFields slice declared directly
// by the top-level struct and by every embedded struct reachable through anonymous
// fields, at any depth. The result is keyed by the index path to each owning struct
// (see indexPathKey), matching structInfo.ForceSendFieldsStructKey. Embedding can be
// arbitrarily deep (e.g. PostgresProject -> PostgresProjectConfig -> ProjectSpec),
// so each level is recorded under its own path rather than collapsed to one index.
func getForceSendFieldsValues(v reflect.Value) map[string]reflect.Value {
	result := make(map[string]reflect.Value)
	collectForceSendFieldsValues(v, nil, result)
	return result
}

func collectForceSendFieldsValues(v reflect.Value, path []int, result map[string]reflect.Value) {
	v = deref(v)
	if !v.IsValid() || v.Kind() != reflect.Struct {
		return
	}

	for i := range v.Type().NumField() {
		field := v.Type().Field(i)
		fieldValue := v.Field(i)

		switch {
		case field.Name == "ForceSendFields" && !field.Anonymous:
			result[indexPathKey(path)] = fieldValue
		case field.Anonymous:
			collectForceSendFieldsValues(fieldValue, append(path, i), result)
		}
	}
}

// indexPathKey renders a reflect index path as a stable map key. The empty path
// (the top-level struct) renders as "".
func indexPathKey(path []int) string {
	parts := make([]string, len(path))
	for i, x := range path {
		parts[i] = strconv.Itoa(x)
	}
	return strings.Join(parts, ".")
}

// deref dereferences a pointer, returning invalid value if nil
func deref(v reflect.Value) reflect.Value {
	if v.Kind() == reflect.Pointer {
		if v.IsNil() {
			return reflect.Value{}
		}
		return v.Elem()
	}
	return v
}

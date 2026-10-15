package convert

import (
	"reflect"
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

	// Tracks fields of API types whose explicitly set zero value is kept on a
	// round trip through the typed representation (see [keepsExplicitZero]).
	KeepExplicitZero map[string]bool
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
		Fields:      make(map[string][]int),
		ForceEmpty:  make(map[string]bool),
		GolangNames: make(map[string]string),

		KeepExplicitZero: make(map[string]bool),
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
			if keepsExplicitZero(styp) {
				out.KeepExplicitZero[name] = true
			}

		}
	}

	return out
}

type FieldValue struct {
	Key   string
	Value reflect.Value
}

func (s *structInfo) FieldValues(v reflect.Value) []FieldValue {
	out := make([]FieldValue, 0, len(s.Fields))

	for _, k := range s.FieldNames {
		fv := fieldByIndex(v, s.Fields[k])

		if fv.IsValid() {
			out = append(out, FieldValue{
				Key:   k,
				Value: fv,
			})
		}
	}

	return out
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

// keepsExplicitZero reports whether fields declared by typ keep an explicitly
// set zero value (e.g. `parent_path: ""`). This holds for API types: the
// sdk-go mirrors and the bundle resource types built on them. The previous SDK
// tracked such values in ForceSendFields; CLI-only types never did.
func keepsExplicitZero(typ reflect.Type) bool {
	pkg := typ.PkgPath()
	return strings.HasPrefix(pkg, "github.com/databricks/cli/libs/sdktypes/") ||
		pkg == "github.com/databricks/cli/bundle/config/resources"
}

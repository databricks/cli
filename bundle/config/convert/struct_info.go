package convert

import (
	"reflect"
	"strings"
	"sync"
)

// structInfo holds the type information we need to efficiently
// convert data from a [config.Value] to a Go struct.
type structInfo struct {
	// Fields maps the JSON-name of the field to the field's index for use with [FieldByIndex].
	Fields map[string][]int
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
	var out = structInfo{
		Fields: make(map[string][]int),
	}

	// Queue holds the indexes of the structs to visit.
	// It is initialized with a single empty slice to visit the top level struct.
	var queue [][]int = [][]int{{}}
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
		for j := 0; j < nf; j++ {
			sf := styp.Field(j)

			// Recurse into anonymous fields.
			if sf.Anonymous {
				queue = append(queue, append(prefix, sf.Index...))
				continue
			}

			name, _, _ := strings.Cut(sf.Tag.Get("json"), ",")
			if name == "" || name == "-" {
				continue
			}

			// Top level fields always take precedence.
			// Therefore, if it is already set, we ignore it.
			if _, ok := out.Fields[name]; ok {
				continue
			}

			out.Fields[name] = append(prefix, sf.Index...)
		}
	}

	return out
}

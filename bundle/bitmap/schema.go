// Package bitmap encodes the presence of every bundle configuration field as a
// single bit. The schema (the ordered list of field paths) is derived by
// reflecting over config.Root and recorded in schema.deflate, which is embedded
// into the binary (acceptance/bundle/bitmap/full lists it). The schema is
// append-only so that bit positions stay stable across releases: a newer binary
// can decode an older bitmap and vice versa.
//
// The bits describe the configuration as the user wrote it, not the result of
// Initialize, so defaults the CLI fills in are not counted (see Builder).
package bitmap

import (
	"bytes"
	"compress/flate"
	"fmt"
	"io"
	"reflect"
	"strings"

	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/libs/structs/structpath"
	"github.com/databricks/cli/libs/structs/structwalk"
)

// EmbeddedSchema returns the schema baked into the binary at build time.
func EmbeddedSchema() []string {
	schema, err := DecompressSchema(schemaDeflate)
	if err != nil {
		// schema.deflate is generated and checked by TestEmbeddedSchemaIsCompleteAndUnique.
		panic(fmt.Sprintf("bitmap: corrupt embedded schema: %s", err))
	}
	return schema
}

// DecompressSchema reverses CompressSchema, returning the field paths.
func DecompressSchema(compressed []byte) ([]string, error) {
	text, err := io.ReadAll(flate.NewReader(bytes.NewReader(compressed)))
	if err != nil {
		return nil, err
	}
	return parseSchema(text), nil
}

// CheckAppendOnly returns an error unless base, an earlier schema, is a prefix
// of schema: every bit keeps its position, and new fields only go at the end.
// Bitmaps already sent are decoded by position, so a moved entry would rename
// their bits.
func CheckAppendOnly(base, schema []string) error {
	for i, p := range base {
		if i >= len(schema) {
			return fmt.Errorf("schema has %d entries, fewer than the %d of the base schema; entries are never removed", len(schema), len(base))
		}
		if schema[i] != p {
			return fmt.Errorf("bit %d is %q in the base schema but %q now; the schema is append-only: keep the base schema as is and append new fields after it", i, p, schema[i])
		}
	}
	return nil
}

// CompressSchema encodes schema text, one field path per line, the way
// schema.deflate stores it: raw DEFLATE. compress/flate gives the same bytes on
// every platform, so the generated file does not depend on who generated it.
func CompressSchema(text []byte) ([]byte, error) {
	var out bytes.Buffer
	w, err := flate.NewWriter(&out, flate.BestCompression)
	if err != nil {
		return nil, err
	}
	if _, err := w.Write(text); err != nil {
		return nil, err
	}
	if err := w.Close(); err != nil {
		return nil, err
	}
	return out.Bytes(), nil
}

// Every bool field has two value entries after its presence entry, so the
// bitmap records true and false apart. A present field with neither value bit
// holds an unresolved variable reference.
const (
	trueSuffix  = "=true"
	falseSuffix = "=false"
)

// listSuffix marks a list element in a schema path.
const listSuffix = "[*]"

// WalkSchema reflects over config.Root and the internal telemetry struct and
// returns the ordered list of field paths, one per field, with map keys
// rendered as ".*" and slice elements as "[*]", and a "=true" and "=false"
// entry after each bool field. The telemetry struct's fields are namespaced
// under "telemetry.". This is the freshly computed schema, in walk order; the
// committed schema.deflate holds every path in it, in its own append order,
// plus any fields since removed.
//
// telemetryType is the internal telemetry struct type (bundle.Telemetry),
// passed in so this package does not import the bundle package.
func WalkSchema(telemetryType reflect.Type) ([]string, error) {
	paths, err := walkTypePaths(reflect.TypeFor[config.Root](), "", isPruned)
	if err != nil {
		return nil, err
	}
	tmPaths, err := telemetryPaths(telemetryType)
	if err != nil {
		return nil, err
	}
	// The walk visits some paths twice: a flattened embed at its parent's path,
	// and a field together with an embedded field it shadows.
	unique, _ := Merge(nil, append(paths, tmPaths...))
	return unique, nil
}

// walkTypePaths walks a struct type and returns each field path, prefixed and
// with pruned subtrees removed. prune may be nil.
func walkTypePaths(t reflect.Type, prefix string, prune func(string) bool) ([]string, error) {
	var paths []string
	err := structwalk.WalkType(t, func(path *structpath.PatternNode, typ reflect.Type, _ *reflect.StructField) bool {
		if path == nil {
			return true
		}
		p := path.String()
		if prune != nil && prune(p) {
			return false
		}
		paths = append(paths, prefix+p)
		if typ.Kind() == reflect.Pointer {
			typ = typ.Elem()
		}
		if typ.Kind() == reflect.Bool || typ.Implements(boolValuesType) {
			paths = append(paths, prefix+p+trueSuffix, prefix+p+falseSuffix)
		}
		return true
	})
	if err != nil {
		return nil, err
	}
	return paths, nil
}

// isPruned reports whether a schema path should be excluded from the bitmap.
func isPruned(p string) bool {
	// targets/environments are per-target overrides that are merged away and set
	// to nil by the SelectTarget mutator, so they are never present at deploy
	// time and would only duplicate the merged tree. __locations is an
	// output-only internal field.
	switch {
	case p == "targets" || strings.HasPrefix(p, "targets."):
		return true
	case p == "environments" || strings.HasPrefix(p, "environments."):
		return true
	case p == "__locations" || strings.HasPrefix(p, "__locations."):
		return true
	// Target selection sets these on every bundle; users do not write them.
	case p == "bundle.target" || p == "bundle.environment":
		return true
	}
	return false
}

// Merge appends to old any paths in fresh that are not already present, keeping
// old's order intact. Paths removed from config.Root remain in the merged schema
// so that existing bit positions never shift. It returns the merged schema and
// the list of newly added paths.
func Merge(old, fresh []string) (merged, added []string) {
	seen := make(map[string]struct{}, len(old))
	for _, p := range old {
		seen[p] = struct{}{}
	}
	merged = append(merged, old...)
	for _, p := range fresh {
		if _, ok := seen[p]; ok {
			continue
		}
		seen[p] = struct{}{}
		merged = append(merged, p)
		added = append(added, p)
	}
	return merged, added
}

func hasEntry(index map[string]int, p string) bool {
	_, ok := index[p]
	return ok
}

// setBit sets the bit of schema path p, if the schema has it.
func setBit(index map[string]int, bits []bool, p string) {
	if i, ok := index[p]; ok {
		bits[i] = true
	}
}

// valueSuffix returns the suffix of the schema entry for a bool value.
func valueSuffix(value bool) string {
	if value {
		return trueSuffix
	}
	return falseSuffix
}

func joinField(pattern, key string) string {
	if pattern == "" {
		return key
	}
	return pattern + "." + key
}

func isSet(index map[string]int, bits []bool, p string) bool {
	i, ok := index[p]
	return ok && bits[i]
}

// indexSchema maps each schema path to its bit position.
func indexSchema(schema []string) map[string]int {
	index := make(map[string]int, len(schema))
	for i, p := range schema {
		index[p] = i
	}
	return index
}

// parseSchema splits schema bytes into field paths. Paths contain no
// whitespace, so splitting on any whitespace also drops the CR of a CRLF
// checkout (Windows) and blank lines, neither of which may become an entry that
// shifts every following bit index.
func parseSchema(b []byte) []string {
	return strings.Fields(string(b))
}

// Package bitmap encodes the presence of every bundle configuration field as a
// single bit. The schema (the ordered list of field paths) is derived by
// reflecting over config.Root and recorded in schema.txt, which is embedded into
// the binary. The schema is append-only so that bit positions stay stable across
// releases: a newer binary can decode an older bitmap and vice versa.
//
// The bits describe the configuration as the user wrote it (bundle.UserConfig),
// not the result of Initialize, so defaults the CLI fills in are not counted.
package bitmap

import (
	"bytes"
	"compress/flate"
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"io"
	"reflect"
	"strings"

	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/structs/structpath"
	"github.com/databricks/cli/libs/structs/structwalk"
)

// magic identifies a bundle telemetry bitmap. The bytes are chosen so they do
// not collide with the leading bytes of the compression formats we wrap the
// payload in (gzip 0x1f 0x8b, zlib 0x78), so a decoder can tell a raw payload
// from a compressed one.
const magic = "DBTB"

// ContextFullBundle marks a bitmap covering the whole user-authored config. Other
// context values are reserved for narrower bitmaps in the future.
const ContextFullBundle uint16 = 0

// headerSize is the size of the fixed header: magic (4) + size (4) + context (2).
const headerSize = 10

// EmbeddedSchema returns the schema baked into the binary at build time.
func EmbeddedSchema() []string {
	return parseSchema(schemaBytes)
}

// telemetryPrefix namespaces the internal telemetry struct's fields in the
// schema so they never collide with config.Root paths.
const telemetryPrefix = "telemetry"

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
// under "telemetry.". This is the freshly computed schema; the
// committed schema.txt is a prefix of it plus any fields since removed.
//
// telemetryType is the internal telemetry struct type (bundle.Telemetry),
// passed in so this package does not import the bundle package.
func WalkSchema(telemetryType reflect.Type) ([]string, error) {
	paths, err := walkTypePaths(reflect.TypeFor[config.Root](), "", isPruned)
	if err != nil {
		return nil, err
	}
	tmPaths, err := walkTypePaths(telemetryType, telemetryPrefix+".", nil)
	if err != nil {
		return nil, err
	}
	return append(paths, tmPaths...), nil
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
		if typ.Kind() == reflect.Bool {
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

// Bits returns a bit per schema entry: true when that config field or telemetry
// flag is present, or for a "=true"/"=false" entry, when the bool holds that
// value. The config is walked as its dyn.Value rather than the typed struct so
// that presence is exactly what the config holds: an interface-typed field
// (variables.*.default) or an empty container ({} or []) counts, and a zero
// value counts only when it is written out. The telemetry struct's paths are
// namespaced under "telemetry.".
func Bits(cfg dyn.Value, telemetry any, schema []string) ([]bool, error) {
	index := indexSchema(schema)

	tm, err := telemetryValue(telemetry)
	if err != nil {
		return nil, err
	}

	bits := make([]bool, len(schema))
	markChildren(cfg, "", index, bits)
	markChildren(tm, telemetryPrefix, index, bits)
	return bits, nil
}

// telemetryValue converts the telemetry struct to a map keyed by Go field name
// that holds only what was measured: a true bool, the value of a non-nil
// *bool, and the elements of a non-empty []bool.
func telemetryValue(telemetry any) (dyn.Value, error) {
	v := reflect.ValueOf(telemetry)
	out := dyn.NewMapping()
	for i := range v.NumField() {
		name := v.Type().Field(i).Name
		switch f := v.Field(i).Interface().(type) {
		case bool:
			if f {
				out.SetLoc(name, nil, dyn.V(true))
			}
		case *bool:
			if f != nil {
				out.SetLoc(name, nil, dyn.V(*f))
			}
		case []bool:
			if len(f) > 0 {
				elems := make([]dyn.Value, len(f))
				for j, value := range f {
					elems[j] = dyn.V(value)
				}
				out.SetLoc(name, nil, dyn.V(elems))
			}
		default:
			return dyn.InvalidValue, fmt.Errorf("telemetry field %s has unsupported type %s", name, v.Field(i).Type())
		}
	}
	return dyn.V(out), nil
}

// markChildren marks every child of v whose schema pattern is known. A map key
// resolves to the struct field of that name when the schema has one and to the
// ".*" wildcard otherwise; a sequence element resolves to "[*]".
func markChildren(v dyn.Value, pattern string, index map[string]int, bits []bool) {
	if m, ok := v.AsMap(); ok {
		for _, pair := range m.Pairs() {
			child := joinField(pattern, pair.Key.MustString())
			if _, ok := index[child]; !ok {
				child = joinField(pattern, "*")
			}
			mark(pair.Value, child, index, bits)
		}
	}
	if s, ok := v.AsSequence(); ok {
		for _, elem := range s {
			mark(elem, pattern+listSuffix, index, bits)
		}
	}
}

// mark sets the bit for a present (non-null) value and recurses into it. A
// pattern outside the schema is not descended into: that covers the pruned
// subtrees and the contents of interface-typed fields, whose own bit is set.
func mark(v dyn.Value, pattern string, index map[string]int, bits []bool) {
	i, ok := index[pattern]
	if !ok || v.Kind() == dyn.KindNil {
		return
	}
	bits[i] = true
	if value, ok := v.AsBool(); ok {
		suffix := falseSuffix
		if value {
			suffix = trueSuffix
		}
		if j, ok := index[pattern+suffix]; ok {
			bits[j] = true
		}
	}
	markChildren(v, pattern, index, bits)
}

func joinField(pattern, key string) string {
	if pattern == "" {
		return key
	}
	return pattern + "." + key
}

// Text renders the bitmap as a "size=N context=C" header followed by one
// "<value> <path>" line per set field in schema order: "T" or "F" for a bool
// field holding that value, "1" for any other present field. A bool shown as "1"
// holds an unresolved variable reference; one shown as both "T" and "F" has both
// values under a wildcard. Unset fields are omitted. bits may be shorter than
// schema (a bitmap from an older CLI) but not longer.
func Text(schema []string, bits []bool, context uint16) []string {
	index := indexSchema(schema[:len(bits)])
	lines := []string{fmt.Sprintf("size=%d context=%d", len(bits), context)}
	for i, p := range schema[:len(bits)] {
		if !bits[i] {
			continue
		}
		if field, ok := strings.CutSuffix(p, trueSuffix); ok {
			lines = append(lines, "T "+field)
			continue
		}
		if field, ok := strings.CutSuffix(p, falseSuffix); ok {
			lines = append(lines, "F "+field)
			continue
		}
		// A bool with a value bit is shown by its value line instead.
		if isSet(index, bits, p+trueSuffix) || isSet(index, bits, p+falseSuffix) {
			continue
		}
		lines = append(lines, "1 "+p)
	}
	return lines
}

// JSON is the JSON rendering of a bitmap. Presence, True and False are trees
// keyed by path segment ("*" for any map key, "[*]" for any list element) with
// true leaves: Presence holds every set field, True and False the bool fields
// seen with that value. A field with set children is a subtree rather than a
// leaf, since a child is only set when its parent is.
type JSON struct {
	Size     int            `json:"size"`
	Context  uint16         `json:"context"`
	Presence map[string]any `json:"presence"`
	True     map[string]any `json:"true"`
	False    map[string]any `json:"false"`
}

// ToJSON renders bits as JSON. bits may be shorter than schema but not longer.
func ToJSON(schema []string, bits []bool, context uint16) JSON {
	out := JSON{
		Size:     len(bits),
		Context:  context,
		Presence: map[string]any{},
		True:     map[string]any{},
		False:    map[string]any{},
	}
	for i, p := range schema[:len(bits)] {
		if !bits[i] {
			continue
		}
		tree := out.Presence
		if field, ok := strings.CutSuffix(p, trueSuffix); ok {
			tree, p = out.True, field
		} else if field, ok := strings.CutSuffix(p, falseSuffix); ok {
			tree, p = out.False, field
		}
		insertPath(tree, treeKeys(p))
	}
	return out
}

// treeKeys splits a schema path into its segments: "a.*.b[*].c" becomes
// a, *, b, [*], c. Schema paths only contain field names, ".*" and "[*]".
func treeKeys(p string) []string {
	var keys []string
	for part := range strings.SplitSeq(p, ".") {
		name, lists := strings.CutSuffix(part, listSuffix)
		keys = append(keys, name)
		if lists {
			keys = append(keys, listSuffix)
		}
	}
	return keys
}

// insertPath adds a true leaf at keys, turning a leaf on the way into a subtree.
func insertPath(tree map[string]any, keys []string) {
	for _, k := range keys[:len(keys)-1] {
		child, ok := tree[k].(map[string]any)
		if !ok {
			child = map[string]any{}
			tree[k] = child
		}
		tree = child
	}
	if _, ok := tree[keys[len(keys)-1]]; !ok {
		tree[keys[len(keys)-1]] = true
	}
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

// Encode packs bits into the wire format (magic, size, context, bitmap),
// compresses it with DEFLATE, and base64-encodes the result. DEFLATE is used
// rather than gzip because it has no timestamp, keeping the output deterministic.
func Encode(bits []bool, context uint16) (string, error) {
	n := len(bits)
	raw := make([]byte, headerSize+(n+7)/8)
	copy(raw, magic)
	binary.BigEndian.PutUint32(raw[4:], uint32(n))
	binary.BigEndian.PutUint16(raw[8:], context)
	for i, set := range bits {
		if set {
			raw[headerSize+i/8] |= 1 << uint(7-i%8)
		}
	}

	var buf bytes.Buffer
	w, err := flate.NewWriter(&buf, flate.BestCompression)
	if err != nil {
		return "", err
	}
	if _, err := w.Write(raw); err != nil {
		return "", err
	}
	if err := w.Close(); err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(buf.Bytes()), nil
}

// Decode reverses Encode: it base64-decodes, inflates, validates the header, and
// returns the bits and context.
func Decode(encoded string) (bits []bool, context uint16, err error) {
	compressed, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return nil, 0, fmt.Errorf("base64: %w", err)
	}
	raw, err := io.ReadAll(flate.NewReader(bytes.NewReader(compressed)))
	if err != nil {
		return nil, 0, fmt.Errorf("inflate: %w", err)
	}
	if len(raw) < headerSize {
		return nil, 0, fmt.Errorf("payload too short: %d bytes", len(raw))
	}
	if string(raw[:4]) != magic {
		return nil, 0, fmt.Errorf("bad magic %q", raw[:4])
	}
	n := binary.BigEndian.Uint32(raw[4:])
	context = binary.BigEndian.Uint16(raw[8:])
	// Compare in uint64 so a corrupt size near 2^32 cannot overflow int on 32-bit platforms.
	if uint64(len(raw)-headerSize) != (uint64(n)+7)/8 {
		return nil, 0, fmt.Errorf("payload size mismatch: %d bits, %d bytes", n, len(raw))
	}
	bits = make([]bool, n)
	for i := range bits {
		bits[i] = raw[headerSize+i/8]&(1<<uint(7-i%8)) != 0
	}
	return bits, context, nil
}

// parseSchema splits schema bytes into field paths. Paths contain no
// whitespace, so splitting on any whitespace also drops the CR of a CRLF
// checkout (Windows) and blank lines, neither of which may become an entry that
// shifts every following bit index.
func parseSchema(b []byte) []string {
	return strings.Fields(string(b))
}

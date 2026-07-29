// Package bitmap encodes the presence of every bundle configuration field as a
// single bit. The schema (the ordered list of field paths) is derived by
// reflecting over config.Root and recorded in schema.txt, which is embedded into
// the binary. The schema is append-only so that bit positions stay stable across
// releases: a newer binary can decode an older bitmap and vice versa.
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
	"github.com/databricks/cli/libs/structs/structpath"
	"github.com/databricks/cli/libs/structs/structwalk"
)

// magic identifies a bundle telemetry bitmap. The bytes are chosen so they do
// not collide with the leading bytes of the compression formats we wrap the
// payload in (gzip 0x1f 0x8b, zlib 0x78), so a decoder can tell a raw payload
// from a compressed one.
const magic = "DBTB"

// ContextFullBundle marks a bitmap covering the whole config.Root tree. Other
// context values are reserved for narrower bitmaps in the future.
const ContextFullBundle uint16 = 0

// headerSize is the size of the fixed header: magic (4) + size (4) + context (2).
const headerSize = 10

// EmbeddedSchema returns the schema baked into the binary at build time.
func EmbeddedSchema() []string {
	return splitLines(schemaBytes)
}

// WalkSchema reflects over config.Root and returns the ordered list of field
// paths, one per field, with map keys rendered as ".*" and slice elements as
// "[*]". This is the freshly computed schema; the committed schema.txt is a
// prefix of it plus any fields since removed from config.Root.
func WalkSchema() ([]string, error) {
	var paths []string
	err := structwalk.WalkType(reflect.TypeFor[config.Root](), func(path *structpath.PatternNode, _ reflect.Type, _ *reflect.StructField) bool {
		if path == nil {
			return true
		}
		p := path.String()
		if isPruned(p) {
			return false
		}
		paths = append(paths, p)
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

// Bits walks the loaded configuration and returns a bit per schema entry: true
// when the field (or any descendant of it) is set. A leaf value sets both its
// own bit and every ancestor-prefix bit.
func Bits(cfg config.Root, schema []string) ([]bool, error) {
	index := make(map[string]int, len(schema))
	for i, p := range schema {
		index[p] = i
	}

	bits := make([]bool, len(schema))
	err := structwalk.Walk(cfg, func(path *structpath.PathNode, _ any, _ *reflect.StructField) {
		for _, pattern := range normalizePrefixes(path) {
			if i, ok := index[pattern]; ok {
				bits[i] = true
			}
		}
	})
	if err != nil {
		return nil, err
	}
	return bits, nil
}

// normalizePrefixes maps a concrete leaf path to its schema pattern and returns
// that pattern together with every ancestor prefix, so that setting a leaf also
// marks intermediate nodes (e.g. "resources.jobs.*") as present.
func normalizePrefixes(path *structpath.PathNode) []string {
	segments := path.AsSlice()
	var result []string
	var pattern *structpath.PatternNode
	for _, node := range segments {
		pattern = appendPattern(pattern, node)
		result = append(result, pattern.String())
	}
	return result
}

// appendPattern extends a pattern with the wildcard form of a concrete node:
// a map key or slice index becomes a wildcard, a struct field keeps its name.
func appendPattern(prev *structpath.PatternNode, node *structpath.PathNode) *structpath.PatternNode {
	if _, ok := node.Index(); ok {
		return structpath.NewPatternBracketStar(prev)
	}
	if _, ok := node.BracketString(); ok {
		return structpath.NewPatternDotStar(prev)
	}
	if key, ok := node.DotString(); ok {
		return structpath.NewPatternDotString(prev, key)
	}
	// The walk only produces indices, map keys, and dot-string fields for
	// config.Root, so any other node kind is unexpected.
	panic(fmt.Sprintf("bitmap: unexpected path node %q", node.String()))
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
	if len(raw) != headerSize+(int(n)+7)/8 {
		return nil, 0, fmt.Errorf("payload size mismatch: %d bits, %d bytes", n, len(raw))
	}
	bits = make([]bool, n)
	for i := range bits {
		bits[i] = raw[headerSize+i/8]&(1<<uint(7-i%8)) != 0
	}
	return bits, context, nil
}

// splitLines splits embedded schema bytes into non-empty lines.
func splitLines(b []byte) []string {
	s := strings.TrimRight(string(b), "\n")
	if s == "" {
		return nil
	}
	return strings.Split(s, "\n")
}

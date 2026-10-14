package schema

import (
	"bytes"
	"compress/gzip"
	_ "embed"
	"io"
)

// The schema is ~1 MB of JSON that only `bundle schema` needs, so the binary
// carries it gzip-compressed (~9x smaller). jsonschema.json stays checked in
// as the readable source; TestEmbeddedSchemaMatchesJSON keeps them in sync.
//
//go:embed jsonschema.json.gz
var compressed []byte

// Bytes returns the bundle configuration JSON schema.
func Bytes() ([]byte, error) {
	r, err := gzip.NewReader(bytes.NewReader(compressed))
	if err != nil {
		return nil, err
	}
	return io.ReadAll(r)
}

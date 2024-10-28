package jsonsaver

import (
	"bytes"
	"encoding/json"
)

// The encoder type encapsulates a [json.Encoder] and its target buffer.
// Escaping of HTML characters in the output is disabled.
type encoder struct {
	*json.Encoder
	*bytes.Buffer
}

func newEncoder() encoder {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	return encoder{enc, &buf}
}

func marshalNoEscape(v any) ([]byte, error) {
	enc := newEncoder()
	err := enc.Encode(v)
	return enc.Bytes(), err
}

func marshalIndentNoEscape(v any, prefix, indent string) ([]byte, error) {
	enc := newEncoder()
	enc.SetIndent(prefix, indent)
	err := enc.Encode(v)
	return enc.Bytes(), err
}

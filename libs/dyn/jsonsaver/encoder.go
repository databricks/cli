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

	// By default, json.Encoder escapes HTML characters, converting symbols like '<' to '\u003c'.
	// This behavior helps prevent XSS attacks when JSON is embedded within HTML.
	// However, we disable this feature since we're not dealing with HTML context.
	// Keeping the escapes enabled would result in unnecessary differences when processing JSON payloads
	// that already contain escaped characters.
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

package jsonsaver

import (
	"bytes"
	"fmt"

	"github.com/databricks/cli/libs/dyn"
)

// Marshal is a version of [json.Marshal] for [dyn.Value].
//
// Objects in the output retain the order of keys as they appear in the underlying [dyn.Value].
// The output does not escape HTML characters in strings.
func Marshal(v dyn.Value) ([]byte, error) {
	return marshalNoEscape(wrap{v})
}

// MarshalIndent is a version of [json.MarshalIndent] for [dyn.Value].
//
// Objects in the output retain the order of keys as they appear in the underlying [dyn.Value].
// The output does not escape HTML characters in strings.
func MarshalIndent(v dyn.Value, prefix, indent string) ([]byte, error) {
	return marshalIndentNoEscape(wrap{v}, prefix, indent)
}

// Wrapper type for [dyn.Value] to expose the [json.Marshaler] interface.
type wrap struct {
	v dyn.Value
}

// MarshalJSON implements the [json.Marshaler] interface for the [dyn.Value] wrapper type.
func (w wrap) MarshalJSON() ([]byte, error) {
	var buf bytes.Buffer
	if err := marshalValue(&buf, w.v); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// marshalValue recursively writes JSON for a [dyn.Value] to the buffer.
func marshalValue(buf *bytes.Buffer, v dyn.Value) error {
	switch v.Kind() {
	case dyn.KindString, dyn.KindBool, dyn.KindInt, dyn.KindFloat, dyn.KindTime, dyn.KindNil:
		out, err := marshalNoEscape(v.AsAny())
		if err != nil {
			return err
		}

		// The encoder writes a trailing newline, so we need to remove it
		// to avoid adding extra newlines when embedding this JSON.
		out = out[:len(out)-1]
		buf.Write(out)
	case dyn.KindMap:
		buf.WriteByte('{')
		for i, pair := range v.MustMap().Pairs() {
			if i > 0 {
				buf.WriteByte(',')
			}
			// Require keys to be strings.
			if pair.Key.Kind() != dyn.KindString {
				return fmt.Errorf("map key must be a string, got %s", pair.Key.Kind())
			}
			// Marshal the key
			if err := marshalValue(buf, pair.Key); err != nil {
				return err
			}
			buf.WriteByte(':')
			// Marshal the value
			if err := marshalValue(buf, pair.Value); err != nil {
				return err
			}
		}
		buf.WriteByte('}')
	case dyn.KindSequence:
		buf.WriteByte('[')
		for i, item := range v.MustSequence() {
			if i > 0 {
				buf.WriteByte(',')
			}
			if err := marshalValue(buf, item); err != nil {
				return err
			}
		}
		buf.WriteByte(']')
	default:
		return fmt.Errorf("unsupported kind: %d", v.Kind())
	}
	return nil
}

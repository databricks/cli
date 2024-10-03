package jsonloader

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"

	"github.com/databricks/cli/libs/dyn"
)

func LoadJSON(data []byte) (dyn.Value, error) {
	offsets := BuildLineOffsets(data)
	reader := bytes.NewReader(data)
	decoder := json.NewDecoder(reader)

	// Start decoding from the top-level value
	value, err := decodeValue(decoder, offsets)
	if err != nil {
		if err == io.EOF {
			err = fmt.Errorf("unexpected end of JSON input")
		}
		return dyn.InvalidValue, err
	}
	return value, nil
}

func decodeValue(decoder *json.Decoder, offsets []LineOffset) (dyn.Value, error) {
	// Read the next JSON token
	token, err := decoder.Token()
	if err != nil {
		return dyn.InvalidValue, err
	}

	// Get the current byte offset
	offset := decoder.InputOffset()
	location := GetPosition(offset, offsets)

	switch tok := token.(type) {
	case json.Delim:
		if tok == '{' {
			// Decode JSON object
			obj := make(map[string]dyn.Value)
			for decoder.More() {
				// Decode the key
				keyToken, err := decoder.Token()
				if err != nil {
					return dyn.InvalidValue, err
				}
				key, ok := keyToken.(string)
				if !ok {
					return dyn.InvalidValue, fmt.Errorf("expected string for object key")
				}

				// Decode the value recursively
				val, err := decodeValue(decoder, offsets)
				if err != nil {
					return dyn.InvalidValue, err
				}

				obj[key] = val
			}
			// Consume the closing '}'
			if _, err := decoder.Token(); err != nil {
				return dyn.InvalidValue, err
			}
			return dyn.NewValue(obj, []dyn.Location{location}), nil
		} else if tok == '[' {
			// Decode JSON array
			var arr []dyn.Value
			for decoder.More() {
				val, err := decodeValue(decoder, offsets)
				if err != nil {
					return dyn.InvalidValue, err
				}
				arr = append(arr, val)
			}
			// Consume the closing ']'
			if _, err := decoder.Token(); err != nil {
				return dyn.InvalidValue, err
			}
			return dyn.NewValue(arr, []dyn.Location{location}), nil
		}
	default:
		// Primitive types: string, number, bool, or null
		return dyn.NewValue(tok, []dyn.Location{location}), nil
	}

	return dyn.InvalidValue, fmt.Errorf("unexpected token: %v", token)
}

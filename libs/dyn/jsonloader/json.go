package jsonloader

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"

	"github.com/databricks/cli/libs/dyn"
)

func LoadJSON(data []byte, source string) (dyn.Value, error) {
	offset := BuildLineOffsets(data)
	offset.SetSource(source)

	reader := bytes.NewReader(data)
	decoder := json.NewDecoder(reader)

	// Start decoding from the top-level value
	value, err := decodeValue(decoder, &offset)
	if err != nil {
		if err == io.EOF {
			err = fmt.Errorf("unexpected end of JSON input")
		}
		return dyn.InvalidValue, fmt.Errorf("error decoding JSON at %s: %v", value.Location(), err)
	}
	return value, nil
}

func decodeValue(decoder *json.Decoder, o *Offset) (dyn.Value, error) {
	// Read the next JSON token
	token, err := decoder.Token()
	if err != nil {
		return dyn.InvalidValue, err
	}

	// Get the current byte offset
	offset := decoder.InputOffset()
	location := o.GetPosition(offset)

	switch tok := token.(type) {
	case json.Delim:
		if tok == '{' {
			// Decode JSON object
			obj := dyn.NewMapping()
			for decoder.More() {
				// Decode the key
				keyToken, err := decoder.Token()
				if err != nil {
					return invalidValueWithLocation(decoder, o), err
				}
				key, ok := keyToken.(string)
				if !ok {
					return invalidValueWithLocation(decoder, o), fmt.Errorf("expected string for object key")
				}

				keyVal := dyn.NewValue(key, []dyn.Location{o.GetPosition(decoder.InputOffset())})
				// Decode the value recursively
				val, err := decodeValue(decoder, o)
				if err != nil {
					return invalidValueWithLocation(decoder, o), err
				}

				obj.Set(keyVal, val)
			}
			// Consume the closing '}'
			if _, err := decoder.Token(); err != nil {
				return invalidValueWithLocation(decoder, o), err
			}
			return dyn.NewValue(obj, []dyn.Location{location}), nil
		} else if tok == '[' {
			// Decode JSON array
			var arr []dyn.Value
			for decoder.More() {
				val, err := decodeValue(decoder, o)
				if err != nil {
					return invalidValueWithLocation(decoder, o), err
				}
				arr = append(arr, val)
			}
			// Consume the closing ']'
			if _, err := decoder.Token(); err != nil {
				return invalidValueWithLocation(decoder, o), err
			}
			return dyn.NewValue(arr, []dyn.Location{location}), nil
		}
	default:
		// Primitive types: string, number, bool, or null
		return dyn.NewValue(tok, []dyn.Location{location}), nil
	}

	return invalidValueWithLocation(decoder, o), fmt.Errorf("unexpected token: %v", token)
}

func invalidValueWithLocation(decoder *json.Decoder, o *Offset) dyn.Value {
	location := o.GetPosition(decoder.InputOffset())
	return dyn.InvalidValue.WithLocations([]dyn.Location{location})
}

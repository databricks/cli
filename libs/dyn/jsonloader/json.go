package jsonloader

import (
	"bytes"
	"encoding/json"
	"errors"
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
			err = errors.New("unexpected end of JSON input")
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

	// Get the current byte offset and the location.
	// We will later use this location to store the location of the value in the file
	// For objects and arrays, we will store the location of the opening '{' or '['
	// For primitive types, we will store the location of the value itself (end of the value)
	// We can't reliably calculate the beginning of the value for primitive types because
	// the decoder doesn't provide the offset of the beginning of the value and the value might or might not be quoted.
	offset := decoder.InputOffset()
	location := o.GetPosition(offset)

	switch tok := token.(type) {
	case json.Delim:
		if tok == '{' {
			location = o.GetPosition(offset - 1)
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
					return invalidValueWithLocation(decoder, o), errors.New("expected string for object key")
				}

				// Get the offset of the key by subtracting the length of the key and the '"' character
				keyOffset := decoder.InputOffset() - int64(len(key)+1)
				loc := []dyn.Location{o.GetPosition(keyOffset)}

				// Decode the value recursively
				val, err := decodeValue(decoder, o)
				if err != nil {
					return invalidValueWithLocation(decoder, o), err
				}

				obj.SetLoc(key, loc, val)
			}
			// Consume the closing '}'
			if _, err := decoder.Token(); err != nil {
				return invalidValueWithLocation(decoder, o), err
			}
			return dyn.NewValue(obj, []dyn.Location{location}), nil
		} else if tok == '[' {
			location = o.GetPosition(offset - 1)
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
		return dyn.NewValue(tok, []dyn.Location{location}), nil
	}

	return invalidValueWithLocation(decoder, o), fmt.Errorf("unexpected token: %v", token)
}

func invalidValueWithLocation(decoder *json.Decoder, o *Offset) dyn.Value {
	location := o.GetPosition(decoder.InputOffset())
	return dyn.InvalidValue.WithLocations([]dyn.Location{location})
}

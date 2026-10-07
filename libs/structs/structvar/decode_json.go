package structvar

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"slices"
	"strconv"

	"github.com/databricks/cli/libs/diag"
	"go.yaml.in/yaml/v3"
)

// ParseJSON parses data into a YAML node tree that carries the positions of the values
// in data: the start of an object or array, the first character of a key and the end of
// a scalar (the decoder does not report where a scalar starts).
func ParseJSON(source string, data []byte) (*yaml.Node, error) {
	decoder := json.NewDecoder(bytes.NewReader(data))
	// Use json.Number to avoid losing precision on int64 values above 2^53 (e.g. job and run IDs).
	decoder.UseNumber()

	lines := newLineIndex(data)
	node, err := decodeJSONValue(decoder, lines)
	if err == nil && decoder.More() {
		err = errors.New("unexpected additional content")
	}
	if err != nil {
		if errors.Is(err, io.EOF) {
			err = errors.New("unexpected end of JSON input")
		}
		line, column := lines.position(decoder.InputOffset())
		return nil, fmt.Errorf("error decoding JSON at %s: %v", diag.Location{File: source, Line: line, Column: column}, err)
	}
	return node, nil
}

// lineIndex holds the offsets at which the lines of the input start.
type lineIndex []int64

func newLineIndex(data []byte) lineIndex {
	lines := lineIndex{0}
	for i, b := range data {
		if b == '\n' {
			lines = append(lines, int64(i+1))
		}
	}
	return lines
}

// position returns the 1-based line and column of offset.
func (l lineIndex) position(offset int64) (line, column int) {
	// The line is the last one starting at or before offset.
	i, found := slices.BinarySearch(l, offset)
	if !found {
		i--
	}
	return i + 1, int(offset-l[i]) + 1
}

func newNode(lines lineIndex, offset int64, kind yaml.Kind, tag, value string) *yaml.Node {
	line, column := lines.position(offset)
	return &yaml.Node{Kind: kind, Tag: tag, Value: value, Line: line, Column: column}
}

func decodeJSONValue(decoder *json.Decoder, lines lineIndex) (*yaml.Node, error) {
	token, err := decoder.Token()
	if err != nil {
		return nil, err
	}
	offset := decoder.InputOffset()

	switch tok := token.(type) {
	case json.Delim:
		if tok == '{' {
			node := newNode(lines, offset-1, yaml.MappingNode, "!!map", "")
			for decoder.More() {
				keyToken, err := decoder.Token()
				if err != nil {
					return nil, err
				}
				key, ok := keyToken.(string)
				if !ok {
					return nil, errors.New("expected string for object key")
				}
				// The decoder reports the end of the key: step back over the key and the closing quote.
				keyNode := newNode(lines, decoder.InputOffset()-int64(len(key)+1), yaml.ScalarNode, "!!str", key)
				value, err := decodeJSONValue(decoder, lines)
				if err != nil {
					return nil, err
				}
				node.Content = append(node.Content, keyNode, value)
			}
			// Consume the closing '}'.
			_, err := decoder.Token()
			return node, err
		}
		node := newNode(lines, offset-1, yaml.SequenceNode, "!!seq", "")
		for decoder.More() {
			value, err := decodeJSONValue(decoder, lines)
			if err != nil {
				return nil, err
			}
			node.Content = append(node.Content, value)
		}
		// Consume the closing ']'.
		_, err := decoder.Token()
		return node, err
	case json.Number:
		if _, err := tok.Int64(); err == nil {
			return newNode(lines, offset, yaml.ScalarNode, "!!int", tok.String()), nil
		}
		// Integers that overflow int64 fall back to float64.
		if _, err := tok.Float64(); err != nil {
			return nil, fmt.Errorf("invalid number %q: %w", tok.String(), err)
		}
		return newNode(lines, offset, yaml.ScalarNode, "!!float", tok.String()), nil
	case string:
		return newNode(lines, offset, yaml.ScalarNode, "!!str", tok), nil
	case bool:
		return newNode(lines, offset, yaml.ScalarNode, "!!bool", strconv.FormatBool(tok)), nil
	default:
		return newNode(lines, offset, yaml.ScalarNode, "!!null", "null"), nil
	}
}

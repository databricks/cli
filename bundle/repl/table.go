package repl

import (
	"encoding/json"
	"fmt"

	"github.com/databricks/databricks-sdk-go/service/commands"
)

type field interface {
	Decode(value any) (any, error)
}

type fieldDefinition struct {
	Name     string `json:"name"`
	Type     any    `json:"type"`
	Metadata any    `json:"metadata"`
}

func parseFieldDefinition(m any) (*fieldDefinition, field, error) {
	buf, err := json.Marshal(m)
	if err != nil {
		return nil, nil, err
	}

	var fd fieldDefinition
	err = json.Unmarshal(buf, &fd)
	if err != nil {
		return nil, nil, err
	}

	// The type field may be JSON encoded (top level fields)
	var tm any
	switch fdt := fd.Type.(type) {
	case string:
		err = json.Unmarshal([]byte(fdt), &tm)
		if err != nil {
			// If "Type" is not valid JSON it must be a literal type name.
			tm = fd.Type
		}
	default:
		tm = fd.Type
	}

	// Turn contents of "type" into a field decoder.
	f, err := typeToField(tm)
	if err != nil {
		return nil, nil, err
	}

	return &fd, f, nil
}

type structFieldType struct {
	names  []string
	fields []field
}

func newStructFieldType(fields []map[string]any) (field, error) {
	var out structFieldType

	for i := range fields {
		fi, f, err := parseFieldDefinition(fields[i])
		if err != nil {
			return nil, err
		}

		out.names = append(out.names, fi.Name)
		out.fields = append(out.fields, f)
	}

	return &out, nil
}

func (s *structFieldType) Decode(value any) (any, error) {
	out := make(map[string]any)
	switch v := value.(type) {
	case []any:
		for i := range v {
			if v[i] == nil {
				continue
			}

			vdec, err := s.fields[i].Decode(v[i])
			if err != nil {
				return nil, err
			}

			out[s.names[i]] = vdec
		}
	default:
		panic(fmt.Errorf("expected []any, got %#v (%T)", value, value))
	}
	return out, nil
}

type stringFieldType struct{}

var stringField stringFieldType

func (f stringFieldType) Decode(value any) (any, error) {
	v, ok := value.(string)
	if !ok {
		return nil, fmt.Errorf("expected string, got %#v (%T)", value, value)
	}

	return v, nil
}

type integerFieldType struct{}

var integerField integerFieldType

func (f integerFieldType) Decode(value any) (any, error) {
	var i int64
	switch v := value.(type) {
	case float32:
		i = int64(v)
	case float64:
		i = int64(v)
	default:
		return nil, fmt.Errorf("expected float, got %#v (%T)", value, value)
	}

	return i, nil
}

type booleanFieldType struct{}

var booleanField booleanFieldType

func (f booleanFieldType) Decode(value any) (any, error) {
	var b bool
	switch v := value.(type) {
	case bool:
		b = v
	default:
		return nil, fmt.Errorf("expected bool, got %#v (%T)", value, value)
	}

	return b, nil
}

type mapFieldType struct{}

var mapField mapFieldType

func (f mapFieldType) Decode(value any) (any, error) {
	var m map[string]any
	switch v := value.(type) {
	case map[string]any:
		m = v
	default:
		return nil, fmt.Errorf("expected map[string]any, got %#v (%T)", value, value)
	}

	// Remove nulls (implied if missing).
	for k, v := range m {
		if v == nil {
			delete(m, k)
		}
	}

	return m, nil
}

func typeToField(typ any) (field, error) {
	switch v := typ.(type) {
	case string:
		switch v {
		case "string":
			return stringField, nil
		case "integer", "long", "bigint":
			return integerField, nil
		case "boolean":
			return booleanField, nil
		case "date":
			return stringField, nil
		case "timestamp":
			return stringField, nil
		default:
			panic("handle " + v)
		}
	case map[string]any:
		typ := v["type"].(string)
		switch typ {
		case "map":
			keyTyp := v["keyType"].(string)
			if keyTyp == "string" {
				return mapField, nil
			} else {
				panic("don't know what to do for non-string maps")
			}
		case "struct":
			var fields []map[string]any
			buf, _ := json.Marshal(v["fields"])
			_ = json.Unmarshal(buf, &fields)
			f, err := newStructFieldType(fields)
			if err != nil {
				return nil, err
			}
			return f, nil
		default:
			panic("handle " + typ)
		}
	default:
		panic(fmt.Sprintf("todo %#v", v))
	}
}

func TableToMap(results *commands.Results) ([]any, error) {
	root, err := newStructFieldType(results.Schema)
	if err != nil {
		return nil, err
	}

	var out []any
	for _, row := range results.Data.([]any) {
		row, err := root.Decode(row)
		if err != nil {
			return nil, err
		}
		out = append(out, row)
	}

	return out, nil
}

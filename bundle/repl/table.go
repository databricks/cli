package repl

import (
	"encoding/json"
	"fmt"

	"github.com/databricks/databricks-sdk-go/service/commands"
)

type field interface {
	Decode(value any, out map[string]any) error
}

type stringField struct {
	name string
}

func (f stringField) Decode(value any, out map[string]any) error {
	if value == nil {
		out[f.name] = nil
		return nil
	}

	v, ok := value.(string)
	if !ok {
		return fmt.Errorf("expected string, got %#v (%T)", value, value)
	}

	out[f.name] = v
	return nil
}

type integerField struct {
	name string
}

func (f integerField) Decode(value any, out map[string]any) error {
	if value == nil {
		out[f.name] = nil
		return nil
	}

	var i int64
	switch v := value.(type) {
	case float32:
		i = int64(v)
	case float64:
		i = int64(v)
	default:
		return fmt.Errorf("expected float, got %#v (%T)", value, value)
	}

	out[f.name] = i
	return nil
}

type booleanField struct {
	name string
}

func (f booleanField) Decode(value any, out map[string]any) error {
	if value == nil {
		out[f.name] = nil
		return nil
	}

	var b bool
	switch v := value.(type) {
	case bool:
		b = v
	default:
		return fmt.Errorf("expected bool, got %#v (%T)", value, value)
	}

	out[f.name] = b
	return nil
}

type topLevelField struct {
	field

	Name     string `json:"name"`
	Type     string `json:"type"`
	Metadata string `json:"metadata"`
}

func newTopLevelFieldType(m map[string]any) (*topLevelField, error) {
	buf, err := json.Marshal(m)
	if err != nil {
		return nil, err
	}

	var out topLevelField
	err = json.Unmarshal(buf, &out)
	if err != nil {
		return nil, err
	}

	// The type field may be JSON encoded (top level fields)
	var tm any
	err = json.Unmarshal([]byte(out.Type), &tm)
	if err != nil {
		// If "Type" is not valid JSON it must be a literal type name.
		tm = out.Type
	}

	switch v := tm.(type) {
	case string:
		switch v {
		case "string":
			out.field = &stringField{out.Name}
		case "integer", "long", "bigint":
			out.field = &integerField{out.Name}
		case "boolean":
			out.field = &booleanField{out.Name}
		case "date":
			out.field = &stringField{out.Name}
		case "timestamp":
			out.field = &stringField{out.Name}
		default:
			panic("handle " + v)
		}
	default:
		panic(fmt.Sprintf("todo %#v", v))
	}

	return &out, nil
}

type tableSchema struct {
	fields []field
}

func (s *tableSchema) Decode(value any, out map[string]any) error {
	switch v := value.(type) {
	case []any:
		for i := range v {
			err := s.fields[i].Decode(v[i], out)
			if err != nil {
				return err
			}
		}
	default:
		panic("bad type")
	}
	return nil
}

func (s *tableSchema) DecodeRow(value any) (map[string]any, error) {
	row, ok := value.([]any)
	if !ok {
		return nil, fmt.Errorf("row must be an array")
	}

	if len(s.fields) != len(row) {
		return nil, fmt.Errorf("schema and row have different number of columns (%d and %d)", len(s.fields), len(row))
	}
	out := make(map[string]any)
	for i := range row {
		err := s.fields[i].Decode(row[i], out)
		if err != nil {
			return nil, err
		}
	}
	return out, nil
}

func TableToMap(results *commands.Results) ([]map[string]any, error) {
	data := results.Data.([]any)
	schema := results.Schema

	var ts tableSchema
	for _, f := range schema {
		fs, err := newTopLevelFieldType(f)
		if err != nil {
			panic(err)
		}
		ts.fields = append(ts.fields, fs)
	}

	var out []map[string]any
	for i := range data {
		row, err := ts.DecodeRow(data[i])
		if err != nil {
			return nil, err
		}
		out = append(out, row)
	}

	return out, nil
}

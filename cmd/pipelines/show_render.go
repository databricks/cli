package pipelines

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
)

// marshals a result row as a JSON object with keys in column order.
type orderedRow struct {
	columns []string
	values  []string
}

func (r orderedRow) MarshalJSON() ([]byte, error) {
	var buf bytes.Buffer
	buf.WriteByte('{')
	for i, col := range r.columns {
		if i > 0 {
			buf.WriteByte(',')
		}
		key, err := json.Marshal(col)
		if err != nil {
			return nil, err
		}
		buf.Write(key)
		buf.WriteByte(':')
		var val []byte
		if i < len(r.values) {
			val, err = json.Marshal(r.values[i])
		} else {
			val = []byte("null")
		}
		if err != nil {
			return nil, err
		}
		buf.Write(val)
	}
	buf.WriteByte('}')
	return buf.Bytes(), nil
}

// writes rows as an indented JSON array of column-ordered objects.
func renderJSON(w io.Writer, columns []string, rows [][]string) error {
	objects := make([]orderedRow, len(rows))
	for i, row := range rows {
		objects[i] = orderedRow{columns: columns, values: row}
	}
	output, err := json.MarshalIndent(objects, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal results: %w", err)
	}
	fmt.Fprintf(w, "%s\n", output)
	return nil
}

package generate

import (
	"bytes"
	"encoding/json"
)

func remarshalJSON(data []byte) ([]byte, error) {
	var tmp any
	var err error
	err = json.Unmarshal(data, &tmp)
	if err != nil {
		return nil, err
	}

	// Remarshal the data to ensure its formatting is stable.
	// The result will have alphabetically sorted keys and be indented.
	// HTML escaping is disabled to retain characters such as &, <, and >.
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetIndent("", "  ")
	enc.SetEscapeHTML(false)
	err = enc.Encode(tmp)
	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

package jsonloader

import (
	"encoding/json"

	"github.com/databricks/cli/libs/dyn"
)

func LoadJSON(data []byte) (dyn.Value, error) {
	var root map[string]interface{}
	err := json.Unmarshal(data, &root)
	if err != nil {
		return dyn.InvalidValue, err
	}

	loc := dyn.Location{
		Line:   1,
		Column: 1,
	}
	return newLoader().load(&root, loc)
}

// Package schema produces the JSON schema for ucm.yml.
//
// M0 generates the schema at runtime via libs/jsonschema.FromType. Once the
// shape stabilizes we will pre-compute it into a checked-in jsonschema.json
// (as bundle/schema does) and embed via //go:embed — the Generate function
// will stay as the generator entry point.
package schema

import (
	"encoding/json"
	"fmt"
	"reflect"

	"github.com/databricks/cli/libs/jsonschema"
	"github.com/databricks/cli/ucm/config"
)

// Generate returns the JSON schema for ucm.yml as indented JSON bytes.
func Generate() ([]byte, error) {
	s, err := jsonschema.FromType(reflect.TypeOf(config.Root{}), nil)
	if err != nil {
		return nil, fmt.Errorf("generate ucm schema: %w", err)
	}
	out, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal ucm schema: %w", err)
	}
	return out, nil
}

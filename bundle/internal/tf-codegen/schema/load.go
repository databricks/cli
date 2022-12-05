package schema

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	tfjson "github.com/hashicorp/terraform-json"
)

func (s *Schema) Load(ctx context.Context) (*tfjson.ProviderSchema, error) {
	buf, err := os.ReadFile(s.ProviderSchemaFile)
	if err != nil {
		return nil, err
	}

	var document tfjson.ProviderSchemas
	err = json.Unmarshal(buf, &document)
	if err != nil {
		return nil, err
	}

	err = document.Validate()
	if err != nil {
		return nil, fmt.Errorf("invalid schema: %w", err)
	}

	// Find the databricks provider definition.
	schema, ok := document.Schemas[DatabricksProvider]
	if !ok {
		return nil, fmt.Errorf("schema file doesn't include schema for %s", DatabricksProvider)
	}

	return schema, nil
}

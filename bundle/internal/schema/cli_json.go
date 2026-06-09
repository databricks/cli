package main

import (
	"encoding/json"
	"fmt"
	"os"
)

// cliJSONField is the shape of a single field in .codegen/cli.json.
type cliJSONField struct {
	Description string   `json:"description,omitempty"`
	Enum        []any    `json:"enum,omitempty"`
	Deprecated  bool     `json:"deprecated,omitempty"`
	Preview     string   `json:"preview,omitempty"`
	Ref         string   `json:"ref,omitempty"`
	Behaviors   []string `json:"behaviors,omitempty"`
}

// cliJSONSchema is the shape of a single schema in .codegen/cli.json.
type cliJSONSchema struct {
	Description string                  `json:"description,omitempty"`
	Enum        []any                   `json:"enum,omitempty"`
	Deprecated  bool                    `json:"deprecated,omitempty"`
	Preview     string                  `json:"preview,omitempty"`
	Fields      map[string]cliJSONField `json:"fields,omitempty"`
}

// cliJSONSpec is the top-level shape of .codegen/cli.json that we care about.
type cliJSONSpec struct {
	Schemas map[string]cliJSONSchema `json:"schemas"`
}

// parseCliJSON reads the .codegen/cli.json spec and returns its schema graph
// keyed by SDK type name (e.g. "jobs.JobSettings"). The annotation parser
// matches Go SDK types against these keys directly; refs between schemas are
// stored as bare type names, so no $ref reconstruction is needed.
func parseCliJSON(path string) (map[string]cliJSONSchema, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var cli cliJSONSpec
	err = json.Unmarshal(b, &cli)
	if err != nil {
		return nil, err
	}

	if len(cli.Schemas) == 0 {
		return nil, fmt.Errorf("no schemas found in %s", path)
	}

	return cli.Schemas, nil
}

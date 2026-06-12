package clijson

import (
	"encoding/json"
	"fmt"
	"os"
)

// Parse reads the cli.json file at path and decodes it into the contract types.
// It is the single entry point for consumers (command generation, bundle schema
// annotations, …) so none of them re-implement decoding of the contract.
func Parse(path string) (*CliJSON, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var doc CliJSON
	if err := json.Unmarshal(b, &doc); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}

	return &doc, nil
}

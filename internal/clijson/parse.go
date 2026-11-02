package clijson

import (
	"encoding/json"
	"fmt"
)

// Parse reassembles the sharded cli.json in dir and decodes it into the
// contract types. It is the single entry point for consumers (command
// generation, bundle schema annotations, …) so none of them re-implement
// decoding of the contract.
func Parse(dir string) (*CliJSON, error) {
	b, err := ReadDir(dir)
	if err != nil {
		return nil, err
	}

	var doc CliJSON
	if err := json.Unmarshal(b, &doc); err != nil {
		return nil, fmt.Errorf("parse %s: %w", dir, err)
	}

	return &doc, nil
}

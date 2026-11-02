package clijson

import (
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"os"
)

// ReadFile reads the gzip-compressed cli.json file at path and returns the
// decompressed JSON. The file is committed compressed to keep it under 1MB.
func ReadFile(path string) ([]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	r, err := gzip.NewReader(f)
	if err != nil {
		return nil, fmt.Errorf("decompress %s: %w", path, err)
	}
	defer r.Close()

	b, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("decompress %s: %w", path, err)
	}
	return b, nil
}

// Parse reads the cli.json file at path and decodes it into the contract types.
// It is the single entry point for consumers (command generation, bundle schema
// annotations, …) so none of them re-implement decoding of the contract.
func Parse(path string) (*CliJSON, error) {
	b, err := ReadFile(path)
	if err != nil {
		return nil, err
	}

	var doc CliJSON
	if err := json.Unmarshal(b, &doc); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}

	return &doc, nil
}

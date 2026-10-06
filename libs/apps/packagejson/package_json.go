package packagejson

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

const FileName = "package.json"

// Read decodes package.json in workDir without interpreting its fields.
func Read(workDir string) (map[string]json.RawMessage, error) {
	data, err := os.ReadFile(filepath.Join(workDir, FileName))
	if err != nil {
		return nil, fmt.Errorf("failed to read %s: %w", FileName, err)
	}
	// npm accepts a UTF-8 BOM: https://github.com/npm/json-parse-even-better-errors.
	data = bytes.TrimPrefix(data, []byte("\xef\xbb\xbf"))
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(data, &fields); err != nil {
		return nil, fmt.Errorf("failed to parse %s: %w", FileName, err)
	}
	return fields, nil
}

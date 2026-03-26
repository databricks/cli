package schema_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/databricks/cli/bundle/schema"
	googleschema "github.com/google/jsonschema-go/jsonschema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.yaml.in/yaml/v3"
)

// schemaKeywords is the set of recognized JSON Schema keywords used to distinguish
// schema definitions from intermediate nesting nodes in $defs.
var schemaKeywords = map[string]bool{
	"$anchor": true, "$defs": true, "$id": true, "$ref": true, "$schema": true,
	"additionalItems": true, "additionalProperties": true,
	"allOf": true, "anyOf": true,
	"const": true, "contains": true,
	"default": true, "deprecated": true, "description": true,
	"else": true, "enum": true, "exclusiveMaximum": true, "exclusiveMinimum": true,
	"format": true,
	"if": true, "items": true,
	"maxItems": true, "maxLength": true, "maxProperties": true,
	"maximum": true, "minItems": true, "minLength": true, "minProperties": true,
	"minimum": true, "multipleOf": true,
	"not": true, "oneOf": true,
	"pattern": true, "patternProperties": true, "prefixItems": true, "properties": true,
	"required": true,
	"then": true, "title": true, "type": true,
	"uniqueItems": true,
}

// isSchemaNode returns true if the object is a JSON Schema definition
// (has recognized keywords or is empty, which is the "anything" schema).
func isSchemaNode(obj map[string]any) bool {
	if len(obj) == 0 {
		return true
	}
	for key := range obj {
		if schemaKeywords[key] {
			return true
		}
	}
	return false
}

// flattenDefs flattens the nested $defs object tree into a single-level map.
// Nested path segments are joined with "/" to form flat keys.
// e.g., $defs["github.com"]["databricks"]["resources.Job"] becomes
// $defs["github.com/databricks/resources.Job"].
func flattenDefs(defs map[string]any) map[string]any {
	result := map[string]any{}
	flattenDefsHelper("", defs, result)
	return result
}

func flattenDefsHelper(prefix string, node map[string]any, result map[string]any) {
	for key, value := range node {
		fullKey := key
		if prefix != "" {
			fullKey = prefix + "/" + key
		}

		obj, isObj := value.(map[string]any)
		if !isObj || isSchemaNode(obj) {
			result[fullKey] = value
		} else {
			flattenDefsHelper(fullKey, obj, result)
		}
	}
}

// rewriteRefs recursively walks a JSON value and rewrites all $ref strings
// to use JSON Pointer ~1 escaping for flat $defs keys.
func rewriteRefs(v any) any {
	switch val := v.(type) {
	case map[string]any:
		result := make(map[string]any, len(val))
		for k, child := range val {
			if k == "$ref" {
				if s, ok := child.(string); ok {
					result[k] = rewriteRef(s)
				} else {
					result[k] = child
				}
			} else {
				result[k] = rewriteRefs(child)
			}
		}
		return result
	case []any:
		result := make([]any, len(val))
		for i, item := range val {
			result[i] = rewriteRefs(item)
		}
		return result
	default:
		return v
	}
}

// rewriteRef transforms a $ref from nested JSON Pointer format to flat key format.
// e.g., "#/$defs/github.com/databricks/resources.Job"
// becomes "#/$defs/github.com~1databricks~1resources.Job"
func rewriteRef(ref string) string {
	const prefix = "#/$defs/"
	if !strings.HasPrefix(ref, prefix) {
		return ref
	}
	path := ref[len(prefix):]
	escaped := strings.ReplaceAll(path, "~", "~0")
	escaped = strings.ReplaceAll(escaped, "/", "~1")
	return prefix + escaped
}

// transformSchema flattens nested $defs and rewrites $ref values for compatibility
// with the Google jsonschema-go library which expects flat $defs.
func transformSchema(raw map[string]any) map[string]any {
	if defs, ok := raw["$defs"].(map[string]any); ok {
		raw["$defs"] = flattenDefs(defs)
	}
	return rewriteRefs(raw).(map[string]any)
}

func compileSchema(t *testing.T) *googleschema.Resolved {
	t.Helper()

	var raw map[string]any
	err := json.Unmarshal(schema.Bytes, &raw)
	require.NoError(t, err)

	transformed := transformSchema(raw)

	b, err := json.Marshal(transformed)
	require.NoError(t, err)

	var s googleschema.Schema
	err = json.Unmarshal(b, &s)
	require.NoError(t, err)

	resolved, err := s.Resolve(nil)
	require.NoError(t, err)

	return resolved
}

// loadYAMLAsJSON reads a YAML file and returns it as a JSON-compatible any value.
// The YAML -> JSON roundtrip ensures canonical JSON types (float64, string, bool, nil,
// map[string]any, []any) that the JSON schema validator expects.
func loadYAMLAsJSON(t *testing.T, path string) any {
	t.Helper()

	data, err := os.ReadFile(path)
	require.NoError(t, err)

	var yamlVal any
	err = yaml.Unmarshal(data, &yamlVal)
	require.NoError(t, err)

	jsonBytes, err := json.Marshal(yamlVal)
	require.NoError(t, err)

	var instance any
	err = json.Unmarshal(jsonBytes, &instance)
	require.NoError(t, err)

	return instance
}

func TestSchemaValidatePassCases(t *testing.T) {
	sch := compileSchema(t)

	files, err := filepath.Glob("../internal/schema/testdata/pass/*.yml")
	require.NoError(t, err)
	require.NotEmpty(t, files)

	for _, file := range files {
		t.Run(filepath.Base(file), func(t *testing.T) {
			instance := loadYAMLAsJSON(t, file)
			err := sch.Validate(instance)
			assert.NoError(t, err)
		})
	}
}

func TestSchemaValidateFailCases(t *testing.T) {
	sch := compileSchema(t)

	files, err := filepath.Glob("../internal/schema/testdata/fail/*.yml")
	require.NoError(t, err)
	require.NotEmpty(t, files)

	for _, file := range files {
		t.Run(filepath.Base(file), func(t *testing.T) {
			instance := loadYAMLAsJSON(t, file)
			err := sch.Validate(instance)
			assert.Error(t, err)
		})
	}
}

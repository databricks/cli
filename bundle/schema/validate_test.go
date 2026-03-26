package schema_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/databricks/cli/bundle/schema"
	"github.com/santhosh-tekuri/jsonschema/v6"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.yaml.in/yaml/v3"
)

func compileSchema(t *testing.T) *jsonschema.Schema {
	t.Helper()

	var schemaVal any
	err := json.Unmarshal(schema.Bytes, &schemaVal)
	require.NoError(t, err)

	c := jsonschema.NewCompiler()
	err = c.AddResource("schema.json", schemaVal)
	require.NoError(t, err)

	sch, err := c.Compile("schema.json")
	require.NoError(t, err)

	return sch
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

package schema_test

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/databricks/cli/bundle/schema"
	"github.com/databricks/cli/libs/jsonschema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func walk(defs map[string]any, p ...string) jsonschema.Schema {
	v, ok := defs[p[0]]
	if !ok {
		panic("not found: " + p[0])
	}

	if len(p) == 1 {
		b, err := json.Marshal(v)
		if err != nil {
			panic(err)
		}
		res := jsonschema.Schema{}
		err = json.Unmarshal(b, &res)
		if err != nil {
			panic(err)
		}
		return res
	}

	return walk(v.(map[string]any), p[1:]...)
}

func TestJsonSchema(t *testing.T) {
	s := jsonschema.Schema{}
	err := json.Unmarshal(schema.Bytes, &s)
	require.NoError(t, err)

	// Assert job fields have their descriptions loaded.
	resourceJob := walk(s.Definitions, "github.com", "databricks", "cli", "bundle", "config", "resources.Job")
	fields := []string{"name", "continuous", "tasks", "trigger"}
	for _, field := range fields {
		assert.NotEmpty(t, resourceJob.OneOf[0].Properties[field].Description)
	}

	// Assert descriptions were also loaded for a job task definition.
	jobTask := walk(s.Definitions, "github.com", "databricks", "databricks-sdk-go", "service", "jobs.Task")
	fields = []string{"notebook_task", "spark_jar_task", "spark_python_task", "spark_submit_task", "description", "depends_on", "environment_key", "for_each_task", "existing_cluster_id"}
	for _, field := range fields {
		assert.NotEmpty(t, jobTask.OneOf[0].Properties[field].Description)
	}

	// Assert descriptions are loaded for pipelines
	pipeline := walk(s.Definitions, "github.com", "databricks", "cli", "bundle", "config", "resources.Pipeline")
	fields = []string{"name", "catalog", "clusters", "channel", "continuous", "development"}
	for _, field := range fields {
		assert.NotEmpty(t, pipeline.OneOf[0].Properties[field].Description)
	}

	providers := walk(s.Definitions, "github.com", "databricks", "databricks-sdk-go", "service", "jobs.GitProvider")
	assert.Contains(t, providers.OneOf[0].Enum, "gitHub")
	assert.Contains(t, providers.OneOf[0].Enum, "bitbucketCloud")
	assert.Contains(t, providers.OneOf[0].Enum, "gitHubEnterprise")
	assert.Contains(t, providers.OneOf[0].Enum, "bitbucketServer")
}

func TestJsonSchemaEnumsAreUnique(t *testing.T) {
	var s any
	err := json.Unmarshal(schema.Bytes, &s)
	require.NoError(t, err)

	duplicateEnums := duplicateEnumPaths(s, "")
	assert.Empty(t, duplicateEnums)
}

func duplicateEnumPaths(v any, path string) []string {
	var duplicates []string
	switch v := v.(type) {
	case map[string]any:
		if enum, ok := v["enum"].([]any); ok {
			seen := map[string]struct{}{}
			for _, item := range enum {
				keyBytes, err := json.Marshal(item)
				if err != nil {
					panic(err)
				}
				key := string(keyBytes)
				if _, ok := seen[key]; ok {
					duplicates = append(duplicates, path)
					break
				}
				seen[key] = struct{}{}
			}
		}
		for key, value := range v {
			childPath := key
			if path != "" {
				childPath = fmt.Sprintf("%s/%s", path, key)
			}
			duplicates = append(duplicates, duplicateEnumPaths(value, childPath)...)
		}
	case []any:
		for i, value := range v {
			duplicates = append(duplicates, duplicateEnumPaths(value, fmt.Sprintf("%s[%d]", path, i))...)
		}
	}
	return duplicates
}

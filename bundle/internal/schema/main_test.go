package main

import (
	"io"
	"os"
	"path"
	"reflect"
	"testing"

	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/libs/jsonschema"
	"github.com/ghodss/yaml"
	"github.com/stretchr/testify/assert"
)

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	if err != nil {
		return err
	}

	return out.Close()
}

// Checks whether descriptions are added for new config fields in the annotations.yml file
// If this test fails either manually add descriptions to the `annotations.yml` or do the following:
//  1. run `make schema` from the repository root to add placeholder descriptions
//  2. replace all "PLACEHOLDER" values with the actual descriptions if possible
//  3. run `make schema` again to regenerate the schema with acutal descriptions
func TestRequiredAnnotationsForNewFields(t *testing.T) {
	workdir := t.TempDir()
	annotationsPath := path.Join(workdir, "annotations.yml")
	annotationsOpenApiPath := path.Join(workdir, "annotations_openapi.yml")
	annotationsOpenApiOverridesPath := path.Join(workdir, "annotations_openapi_overrides.yml")

	// Copy existing annotation files from the same folder as this test
	err := copyFile("annotations.yml", annotationsPath)
	assert.NoError(t, err)
	err = copyFile("annotations_openapi.yml", annotationsOpenApiPath)
	assert.NoError(t, err)
	err = copyFile("annotations_openapi_overrides.yml", annotationsOpenApiOverridesPath)
	assert.NoError(t, err)

	generateSchema(workdir, path.Join(t.TempDir(), "schema.json"))

	original, err := os.ReadFile("annotations.yml")
	assert.NoError(t, err)
	copied, err := os.ReadFile(annotationsPath)
	assert.NoError(t, err)
	assert.Equal(t, string(original), string(copied), "Missing JSON-schema descriptions for new config fields in bundle/internal/schema/annotations.yml")
}

// Checks whether types in annotation files are still present in Config type
func TestNoDetachedAnnotations(t *testing.T) {
	files := []string{
		"annotations.yml",
		"annotations_openapi.yml",
		"annotations_openapi_overrides.yml",
	}

	types := map[string]bool{}
	for _, file := range files {
		annotations, err := getAnnotations(file)
		assert.NoError(t, err)
		for k := range annotations {
			types[k] = false
		}
	}

	_, err := jsonschema.FromType(reflect.TypeOf(config.Root{}), []func(reflect.Type, jsonschema.Schema) jsonschema.Schema{
		func(typ reflect.Type, s jsonschema.Schema) jsonschema.Schema {
			delete(types, getPath(typ))
			return s
		},
	})
	assert.NoError(t, err)

	for typ := range types {
		t.Errorf("Type `%s` in annotations file is not found in `root.Config` type", typ)
	}
	assert.Empty(t, types, "Detached annotations found, regenerate schema and check for package path changes")
}

func getAnnotations(path string) (annotationFile, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var data annotationFile
	err = yaml.Unmarshal(b, &data)
	return data, err
}

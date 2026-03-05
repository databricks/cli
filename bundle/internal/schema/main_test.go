package main

import (
	"bytes"
	"io"
	"os"
	"path"
	"reflect"
	"strings"
	"testing"

	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/internal/annotation"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/dyn/merge"
	"github.com/databricks/cli/libs/dyn/yamlloader"
	"github.com/databricks/cli/libs/jsonschema"
	"github.com/stretchr/testify/assert"
	"go.yaml.in/yaml/v3"
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

// Checks whether descriptions are added for new config fields in the annotations.yml file.
// This test requires the DATABRICKS_OPENAPI_SPEC environment variable to be set to
// determine which fields get descriptions from the OpenAPI spec vs which need manual entries.
//
// If this test fails either manually add descriptions to the `annotations.yml` or do the following:
//  1. Run `make schema` from the repository root to add placeholder descriptions
//  2. Replace all "PLACEHOLDER" values with the actual descriptions if possible
//  3. Run `make schema` again to regenerate the schema with actual descriptions
func TestRequiredAnnotationsForNewFields(t *testing.T) {
	if os.Getenv("DATABRICKS_OPENAPI_SPEC") == "" {
		t.Skip("DATABRICKS_OPENAPI_SPEC not set, skipping annotation completeness check")
	}

	workdir := t.TempDir()
	annotationsPath := path.Join(workdir, "annotations.yml")

	// Copy existing annotation file from the same folder as this test
	err := copyFile("annotations.yml", annotationsPath)
	assert.NoError(t, err)

	generateSchema(workdir, path.Join(t.TempDir(), "schema.json"), false)

	originalFile, err := os.ReadFile("annotations.yml")
	assert.NoError(t, err)
	currentFile, err := os.ReadFile(annotationsPath)
	assert.NoError(t, err)
	original, err := yamlloader.LoadYAML("", bytes.NewBuffer(originalFile))
	assert.NoError(t, err)
	current, err := yamlloader.LoadYAML("", bytes.NewBuffer(currentFile))
	assert.NoError(t, err)

	// Collect added paths.
	var updatedFieldPaths []string
	_, err = merge.Override(original, current, merge.OverrideVisitor{
		VisitInsert: func(basePath dyn.Path, right dyn.Value) (dyn.Value, error) {
			updatedFieldPaths = append(updatedFieldPaths, basePath.String())
			return right, nil
		},
	})
	assert.NoError(t, err)
	assert.Empty(t, updatedFieldPaths, "Missing JSON-schema descriptions for new config fields in bundle/internal/schema/annotations.yml:\n%s", strings.Join(updatedFieldPaths, "\n"))
}

// Checks whether types in the annotations file are still present in the Config type.
func TestNoDetachedAnnotations(t *testing.T) {
	bundlePaths := map[string]bool{}
	annotations, err := getAnnotations("annotations.yml")
	assert.NoError(t, err)
	for k := range annotations {
		bundlePaths[k] = false
	}

	// Use the path mapping to convert Go type paths to bundle paths during the walk.
	m := buildPathMapping()

	_, err = jsonschema.FromType(reflect.TypeOf(config.Root{}), []func(reflect.Type, jsonschema.Schema) jsonschema.Schema{
		func(typ reflect.Type, s jsonschema.Schema) jsonschema.Schema {
			typePath := getPath(typ)
			// Match by bundle path (for entries using bundle path keys)
			if bp, ok := m.typeToBundlePath[typePath]; ok {
				delete(bundlePaths, bp)
			}
			// Also match by Go type path (for entries using Go type path keys)
			delete(bundlePaths, typePath)
			return s
		},
	})
	assert.NoError(t, err)

	for bp := range bundlePaths {
		t.Errorf("Type `%s` in annotations file is not found in `root.Config` type", bp)
	}
	assert.Empty(t, bundlePaths, "Detached annotations found, regenerate schema and check for package path changes")
}

func getAnnotations(path string) (annotation.File, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var data annotation.File
	err = yaml.Unmarshal(b, &data)
	return data, err
}

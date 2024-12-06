package main

import (
	"io"
	"os"
	"path"
	"runtime"
	"testing"

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
	if runtime.GOOS == "windows" {
		t.Skip()
	}
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

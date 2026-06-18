package main

import (
	"bytes"
	"io"
	"os"
	"path"
	"strings"
	"testing"

	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/dyn/merge"
	"github.com/databricks/cli/libs/dyn/yamlloader"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const cliJSONPath = "../../../.codegen/cli.json"

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
// If this test fails:
//  1. run `./task generate-schema` from the repository root to add placeholder descriptions
//  2. replace all "PLACEHOLDER" values with the actual descriptions if possible
//  3. run `./task generate-schema` again to regenerate the schema with actual descriptions
func TestRequiredAnnotationsForNewFields(t *testing.T) {
	workdir := t.TempDir()
	annotationsPath := path.Join(workdir, "annotations.yml")

	err := copyFile("annotations.yml", annotationsPath)
	require.NoError(t, err)

	generateSchema(workdir, path.Join(t.TempDir(), "schema.json"), cliJSONPath, false)

	originalFile, err := os.ReadFile("annotations.yml")
	require.NoError(t, err)
	currentFile, err := os.ReadFile(annotationsPath)
	require.NoError(t, err)
	original, err := yamlloader.LoadYAML("", bytes.NewBuffer(originalFile))
	require.NoError(t, err)
	current, err := yamlloader.LoadYAML("", bytes.NewBuffer(currentFile))
	require.NoError(t, err)

	// Regenerating from the committed file must be a no-op: no new placeholders
	// (a new undocumented config field) and no deletes/updates (stale
	// placeholders not yet pruned). VisitDelete/VisitUpdate must be set or
	// Override panics on any change.
	var addedFieldPaths []string
	var changedFieldPaths []string
	_, err = merge.Override(original, current, merge.OverrideVisitor{
		VisitInsert: func(basePath dyn.Path, right dyn.Value) (dyn.Value, error) {
			addedFieldPaths = append(addedFieldPaths, basePath.String())
			return right, nil
		},
		VisitDelete: func(basePath dyn.Path, left dyn.Value) error {
			changedFieldPaths = append(changedFieldPaths, basePath.String())
			return nil
		},
		VisitUpdate: func(basePath dyn.Path, left, right dyn.Value) (dyn.Value, error) {
			changedFieldPaths = append(changedFieldPaths, basePath.String())
			return right, nil
		},
	})
	assert.NoError(t, err)
	assert.Empty(t, addedFieldPaths, "Missing JSON-schema descriptions for new config fields in bundle/internal/schema/annotations.yml:\n%s", strings.Join(addedFieldPaths, "\n"))
	assert.Empty(t, changedFieldPaths, "annotations.yml is out of sync; run `./task generate-schema` and commit the result:\n%s", strings.Join(changedFieldPaths, "\n"))
}

// Checks that the annotations file only contains entries that match the
// current bundle configuration structure.
func TestNoDetachedAnnotations(t *testing.T) {
	g, err := configTypeGraph()
	require.NoError(t, err)

	_, unknown, err := loadAnnotationsFile("annotations.yml", g)
	require.NoError(t, err)
	assert.Empty(t, unknown, "Detached annotations found; run `./task generate-schema` to drop them")
}

package main

import (
	"bytes"
	"io"
	"os"
	"path"
	"reflect"
	"strings"
	"testing"

	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/dyn/merge"
	"github.com/databricks/cli/libs/dyn/yamlloader"
	"github.com/databricks/cli/libs/jsonschema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const cliJSONPath = "../../../.codegen/cli.json"

func TestJobRunIdempotencyTokenIsNotInSchema(t *testing.T) {
	s := jsonschema.Schema{
		Properties: map[string]*jsonschema.Schema{
			"idempotency_token": {},
			"job_id":            {},
		},
	}

	s = removeJobsFields(reflect.TypeFor[resources.JobRun](), s)

	assert.NotContains(t, s.Properties, "idempotency_token")
	assert.Contains(t, s.Properties, "job_id")
}

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

// regenerateAnnotationFiles regenerates the annotation files from the committed ones into
// a temp dir and returns, per file, the tree paths that were added and the
// paths that were changed or removed.
func regenerateAnnotationFiles(t *testing.T) (added, changed map[string][]string) {
	workdir := t.TempDir()
	for _, name := range []string{annotationsFileName, generatedFileName} {
		require.NoError(t, copyFile(name, path.Join(workdir, name)))
	}

	generateSchema(workdir, path.Join(t.TempDir(), "schema.json"), cliJSONPath, false)

	added = map[string][]string{}
	changed = map[string][]string{}
	for _, name := range []string{annotationsFileName, generatedFileName} {
		originalFile, err := os.ReadFile(name)
		require.NoError(t, err)
		currentFile, err := os.ReadFile(path.Join(workdir, name))
		require.NoError(t, err)
		original, err := yamlloader.LoadYAML("", bytes.NewBuffer(originalFile))
		require.NoError(t, err)
		current, err := yamlloader.LoadYAML("", bytes.NewBuffer(currentFile))
		require.NoError(t, err)

		// VisitDelete/VisitUpdate must be set or Override panics on any change.
		_, err = merge.Override(original, current, merge.OverrideVisitor{
			VisitInsert: func(basePath dyn.Path, right dyn.Value) (dyn.Value, error) {
				added[name] = append(added[name], basePath.String())
				return right, nil
			},
			VisitDelete: func(basePath dyn.Path, left dyn.Value) error {
				changed[name] = append(changed[name], basePath.String())
				return nil
			},
			VisitUpdate: func(basePath dyn.Path, left, right dyn.Value) (dyn.Value, error) {
				changed[name] = append(changed[name], basePath.String())
				return right, nil
			},
		})
		require.NoError(t, err)
	}
	return added, changed
}

// Checks whether descriptions are added for new config fields.
// If this test fails:
//  1. run `./task generate-schema` from the repository root; new undocumented
//     fields appear as PLACEHOLDER entries in annotations.generated.yml
//  2. document them in annotations.yml at the same tree position if possible
//  3. run `./task generate-schema` again to regenerate the schema with the
//     new descriptions and drop the entries from annotations.generated.yml
func TestRequiredAnnotationsForNewFields(t *testing.T) {
	added, changed := regenerateAnnotationFiles(t)

	assert.Empty(t, added[generatedFileName], "Missing JSON-schema descriptions for new config fields; run `./task generate-schema` and document them in bundle/internal/schema/annotations.yml:\n%s", strings.Join(added[generatedFileName], "\n"))
	assert.Empty(t, changed[generatedFileName], "annotations.generated.yml is out of sync; run `./task generate-schema` and commit the result:\n%s", strings.Join(changed[generatedFileName], "\n"))

	// Regenerating from the committed hand-authored file must be a no-op.
	assert.Empty(t, added[annotationsFileName], "annotations.yml is out of sync; run `./task generate-schema` and commit the result:\n%s", strings.Join(added[annotationsFileName], "\n"))
	assert.Empty(t, changed[annotationsFileName], "annotations.yml is out of sync; run `./task generate-schema` and commit the result:\n%s", strings.Join(changed[annotationsFileName], "\n"))
}

// Checks that the annotations file only contains entries that match the
// current bundle configuration structure.
func TestNoDetachedAnnotations(t *testing.T) {
	g, err := configTypeGraph()
	require.NoError(t, err)

	for _, name := range []string{annotationsFileName, generatedFileName} {
		_, unknown, err := loadAnnotationsFile(name, g)
		require.NoError(t, err)
		assert.Empty(t, unknown, "Detached annotations found in %s; run `./task generate-schema` to drop them", name)
	}
}

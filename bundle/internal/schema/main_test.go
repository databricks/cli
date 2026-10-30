package main

import (
	"io"
	"maps"
	"os"
	"path"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/libs/jsonschema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.yaml.in/yaml/v3"
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
	var original, current any
	require.NoError(t, yaml.Unmarshal(originalFile, &original))
	require.NoError(t, yaml.Unmarshal(currentFile, &current))

	// Regenerating from the committed file must be a no-op: no new placeholders
	// (a new undocumented config field) and no deletes/updates (stale
	// placeholders not yet pruned).
	var addedFieldPaths []string
	var changedFieldPaths []string
	diffYAML("", original, current, &addedFieldPaths, &changedFieldPaths)
	assert.Empty(t, addedFieldPaths, "Missing JSON-schema descriptions for new config fields in bundle/internal/schema/annotations.yml:\n%s", strings.Join(addedFieldPaths, "\n"))
	assert.Empty(t, changedFieldPaths, "annotations.yml is out of sync; run `./task generate-schema` and commit the result:\n%s", strings.Join(changedFieldPaths, "\n"))
}

// diffYAML records the paths of the values added to, and removed from or changed in, left to get right.
func diffYAML(path string, left, right any, added, changed *[]string) {
	l, lok := left.(map[string]any)
	r, rok := right.(map[string]any)
	if !lok || !rok {
		if !reflect.DeepEqual(left, right) {
			*changed = append(*changed, path)
		}
		return
	}
	for _, k := range slices.Sorted(maps.Keys(l)) {
		if _, ok := r[k]; !ok {
			*changed = append(*changed, strings.TrimPrefix(path+"."+k, "."))
		}
	}
	for _, k := range slices.Sorted(maps.Keys(r)) {
		p := strings.TrimPrefix(path+"."+k, ".")
		if lv, ok := l[k]; ok {
			diffYAML(p, lv, r[k], added, changed)
		} else {
			*added = append(*added, p)
		}
	}
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

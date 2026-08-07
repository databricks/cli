package main

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"path"
	"strings"
	"testing"

	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/dyn/merge"
	"github.com/databricks/cli/libs/dyn/yamlloader"
	"github.com/databricks/cli/libs/jsonschema"
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

// buildTestSchema generates the in-memory bundle schema the same way the
// generator does, against a throwaway copy of the committed annotations file
// (buildSchema rewrites it in place).
func buildTestSchema(t *testing.T, docsMode bool) jsonschema.Schema {
	t.Helper()
	workdir := t.TempDir()
	require.NoError(t, copyFile("annotations.yml", path.Join(workdir, "annotations.yml")))
	s, err := buildSchema(workdir, cliJSONPath, docsMode)
	require.NoError(t, err)
	return s
}

func mustMarshalSchema(t *testing.T, s jsonschema.Schema) string {
	t.Helper()
	b, err := json.Marshal(s)
	require.NoError(t, err)
	return string(b)
}

// The docs schema is no longer generated or checked in on main; it is built
// only on release and published to the docgen branch. These tests exercise the
// docsMode build path so a regression in it fails CI here instead of surfacing
// as missing/incorrect fields in the published docs.

// Docs mode must drop the interpolation-pattern transform, so the published
// docs schema shows plain field types rather than the runtime `${...}` unions.
func TestBuildDocsSchemaOmitsInterpolationPatterns(t *testing.T) {
	docs := buildTestSchema(t, true)
	runtime := buildTestSchema(t, false)

	require.NotEmpty(t, docs.Properties, "docs schema has no root properties")
	require.NotEmpty(t, docs.Definitions, "docs schema has no $defs")

	// Derive the marker from the generator's own helper so the assertion can't
	// drift from what it emits. json.Marshal yields the quoted, escaped regex
	// exactly as it appears in the schema; strip the surrounding quotes.
	encoded, err := json.Marshal(interpolationPattern("bundle"))
	require.NoError(t, err)
	marker := string(encoded[1 : len(encoded)-1])

	assert.Contains(t, mustMarshalSchema(t, runtime), marker, "runtime schema should contain interpolation patterns")
	assert.NotContains(t, mustMarshalSchema(t, docs), marker, "docs schema must omit interpolation patterns")
}

// computeSinceVersions emits keys via flattenSchema; addSinceVersionToSchema
// consumes the same key format. This feeds every such key a sentinel version
// and asserts it lands on both root properties and nested $defs, guarding the
// two walks against drifting apart, which would silently drop x-since-version
// from the published docs schema.
func TestDocsSchemaSinceVersionRoundTrip(t *testing.T) {
	s := buildTestSchema(t, true)

	var raw map[string]any
	require.NoError(t, json.Unmarshal([]byte(mustMarshalSchema(t, s)), &raw))
	fields := flattenSchema(raw)
	require.NotEmpty(t, fields)

	const sentinel = "v9.9.9"
	sinceVersions := make(map[string]string, len(fields))
	for key := range fields {
		sinceVersions[key] = sentinel
	}

	addSinceVersionToSchema(&s, sinceVersions)

	// Every root property key is in fields, so all must be stamped.
	require.NotEmpty(t, s.Properties)
	for name, prop := range s.Properties {
		assert.Equal(t, sentinel, prop.SinceVersion, "root property %q missing x-since-version", name)
	}

	// $defs are stamped by walkDefinitions. If its type assertions stop matching
	// the schema structure, nested fields silently keep an empty version, so
	// assert the stamp reached well beyond the root properties.
	needle := `"x-since-version":"` + sentinel + `"`
	stamped := strings.Count(mustMarshalSchema(t, s), needle)
	assert.Greater(t, stamped, len(s.Properties), "x-since-version should reach $defs fields, not only root properties")
}

package main

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/databricks/cli/bundle/internal/annotation"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAnnotationsFileRoundTrip(t *testing.T) {
	g, err := newTypeGraph(reflect.TypeFor[tgRoot]())
	require.NoError(t, err)

	data := annotation.File{
		getPath(reflect.TypeFor[tgRoot]()): {
			Fields: map[string]annotation.Descriptor{
				"first":   {Description: "First field."},
				"by_name": {Description: "Map field.", MarkdownDescription: "Map field. See [_](/foo.md)."},
			},
		},
		getPath(reflect.TypeFor[tgNested]()): {
			Self: annotation.Descriptor{Description: "A nested type."},
			Fields: map[string]annotation.Descriptor{
				"description": {Description: annotation.Placeholder},
				"again":       {Description: "Recursive field."},
			},
		},
		getPath(reflect.TypeFor[tgMode]()): {
			Self: annotation.Descriptor{Enum: []any{"a", "b"}},
		},
		getPath(reflect.TypeFor[tgItem]()): {
			Fields: map[string]annotation.Descriptor{
				"inner": {Description: "Inner docs."},
			},
		},
	}

	dir := t.TempDir()
	detached, err := saveAnnotationsDir(dir, handSuffix, data, g)
	require.NoError(t, err)
	assert.Empty(t, detached)

	// tgRoot has no "resources" field, so everything lands in root.yml.
	names, err := listLayer(dir, handSuffix)
	require.NoError(t, err)
	assert.Equal(t, []string{rootFileName}, names)

	b, err := os.ReadFile(filepath.Join(dir, "root.yml"))
	require.NoError(t, err)

	// Keys are emitted alphabetically; tgNested expands under `first` (its
	// canonical position in declaration order), so `by_name`, which resolves
	// to the same type, carries no `$type` or `$fields` keys. `plain` has no
	// content and is omitted entirely.
	expected := annotationsFileHeader + `by_name:
  "description": |-
    Map field.
  "markdown_description": |-
    Map field. See [_](/foo.md).
first:
  "description": |-
    First field.
  "$type":
    "description": |-
      A nested type.
  "$fields":
    "again":
      "description": |-
        Recursive field.
    "description":
      "description": |-
        PLACEHOLDER
items:
  "$fields":
    "inner":
      "description": |-
        Inner docs.
mode:
  "$type":
    "enum":
      - |-
        a
      - |-
        b
`
	assert.Equal(t, expected, string(b))

	loaded, unknown, err := loadAnnotationsDir(dir, handSuffix, g)
	require.NoError(t, err)
	assert.Empty(t, unknown)
	assert.Equal(t, data, loaded)

	// Saving the loaded data again must be byte-identical.
	dir2 := t.TempDir()
	_, err = saveAnnotationsDir(dir2, handSuffix, loaded, g)
	require.NoError(t, err)
	b2, err := os.ReadFile(filepath.Join(dir2, "root.yml"))
	require.NoError(t, err)
	assert.Equal(t, string(b), string(b2))
}

func TestAnnotationsDirSplitsResources(t *testing.T) {
	g, err := newTypeGraph(reflect.TypeFor[tgSplitRoot]())
	require.NoError(t, err)

	data := annotation.File{
		getPath(reflect.TypeFor[tgSplitRoot]()): {
			Fields: map[string]annotation.Descriptor{
				"resources": {Description: "All resources."},
				"other":     {Description: "Not a resource."},
			},
		},
		getPath(reflect.TypeFor[tgResources]()): {
			Fields: map[string]annotation.Descriptor{
				"jobs": {Description: "Jobs."},
			},
		},
		getPath(reflect.TypeFor[tgItem]()): {
			Fields: map[string]annotation.Descriptor{
				"inner": {Description: "Inner docs."},
			},
		},
	}

	dir := t.TempDir()
	// A stale file of this layer is removed; the other layer is untouched.
	require.NoError(t, os.WriteFile(filepath.Join(dir, "gone.yml"), []byte("description: stale\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "jobs.generated.yml"), []byte("description: PLACEHOLDER\n"), 0o644))

	overrides := annotation.File{
		getPath(reflect.TypeFor[tgResources]()): {
			Fields: map[string]annotation.Descriptor{
				"pipelines": {Description: "Pipelines.", DeprecationMessage: "Use jobs."},
			},
		},
	}

	detached, err := saveAnnotationsDir(dir, handSuffix, data, g)
	require.NoError(t, err)
	assert.Empty(t, detached)
	detached, err = saveAnnotationsDir(dir, overridesSuffix, overrides, g)
	require.NoError(t, err)
	assert.Empty(t, detached)

	// Each layer is sliced the same way and cleans up only its own files.
	names, err := listLayer(dir, handSuffix)
	require.NoError(t, err)
	assert.Equal(t, []string{"jobs", "root"}, names)
	names, err = listLayer(dir, overridesSuffix)
	require.NoError(t, err)
	assert.Equal(t, []string{"pipelines"}, names)
	assert.FileExists(t, filepath.Join(dir, "jobs.generated.yml"))

	b, err := os.ReadFile(filepath.Join(dir, "pipelines.overrides.yml"))
	require.NoError(t, err)
	assert.Equal(t, overridesFileHeader+`description: |-
  Pipelines.
deprecation_message: |-
  Use jobs.
`, string(b))

	b, err = os.ReadFile(filepath.Join(dir, "root.yml"))
	require.NoError(t, err)
	assert.Equal(t, annotationsFileHeader+`other:
  "description": |-
    Not a resource.
resources:
  "description": |-
    All resources.
`, string(b))

	// tgItem expands under jobs (first in declaration order), so pipelines,
	// which resolves to the same type, would carry only its own descriptor.
	// The saver quotes keys below the styled top level only, so a resource
	// file's node keys render bare.
	b, err = os.ReadFile(filepath.Join(dir, "jobs.yml"))
	require.NoError(t, err)
	assert.Equal(t, annotationsFileHeader+`description: |-
  Jobs.
$fields:
  "inner":
    "description": |-
      Inner docs.
`, string(b))

	// Each layer loads back independently.
	loaded, unknown, err := loadAnnotationsDir(dir, handSuffix, g)
	require.NoError(t, err)
	assert.Empty(t, unknown)
	assert.Equal(t, data, loaded)
	loaded, unknown, err = loadAnnotationsDir(dir, overridesSuffix, g)
	require.NoError(t, err)
	assert.Empty(t, unknown)
	assert.Equal(t, overrides, loaded)

	// The generated layer loads independently and reports files for unknown
	// resources.
	require.NoError(t, os.WriteFile(filepath.Join(dir, "nope.generated.yml"), []byte("description: PLACEHOLDER\n"), 0o644))
	generated, unknown, err := loadAnnotationsDir(dir, generatedSuffix, g)
	require.NoError(t, err)
	assert.Equal(t, []string{"nope.generated.yml"}, unknown)
	assert.Equal(t, annotation.Placeholder, generated[getPath(reflect.TypeFor[tgResources]())].Fields["jobs"].Description)
}

func TestAnnotationsFileUnknownEntries(t *testing.T) {
	g, err := newTypeGraph(reflect.TypeFor[tgRoot]())
	require.NoError(t, err)

	dir := t.TempDir()
	err = os.WriteFile(filepath.Join(dir, "root.yml"), []byte(`first:
  description: |-
    Valid entry.
  bogus_key: |-
    Not a descriptor key.
  type:
    description: |-
      Unprefixed "type" is not the structural key, so it is unknown here.
  $fields:
    no_such_field:
      description: |-
        Field does not exist.
plain:
  $fields:
    nested:
      description: |-
        Primitive fields have no nested fields.
  $type:
    description: |-
      Primitive fields have no type docs.
`), 0o644)
	require.NoError(t, err)

	data, unknown, err := loadAnnotationsDir(dir, handSuffix, g)
	require.NoError(t, err)
	assert.Equal(t, []string{
		"first.bogus_key",
		"first.type",
		"first.$fields.no_such_field",
		"plain.$fields",
		"plain.$type",
	}, unknown)

	// The valid entry is still loaded.
	assert.Equal(t, "Valid entry.", data[getPath(reflect.TypeFor[tgRoot]())].Fields["first"].Description)
}

func TestAnnotationsFileDetachedEntries(t *testing.T) {
	g, err := newTypeGraph(reflect.TypeFor[tgRoot]())
	require.NoError(t, err)

	data := annotation.File{
		getPath(reflect.TypeFor[tgRoot]()): {
			Fields: map[string]annotation.Descriptor{
				"no_such_field": {Description: "Stale."},
			},
		},
		"github.com/databricks/cli/no/such.Type": {
			Self: annotation.Descriptor{Description: "Stale type."},
			Fields: map[string]annotation.Descriptor{
				"field": {Description: "Stale."},
			},
		},
	}

	detached, err := saveAnnotationsDir(t.TempDir(), handSuffix, data, g)
	require.NoError(t, err)
	assert.Equal(t, []string{
		"github.com/databricks/cli/bundle/internal/schema.tgRoot: no_such_field",
		"github.com/databricks/cli/no/such.Type: (type)",
		"github.com/databricks/cli/no/such.Type: field",
	}, detached)
}

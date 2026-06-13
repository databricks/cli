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

	path := filepath.Join(t.TempDir(), "annotations.yml")
	detached, err := saveAnnotationsFile(path, data, g)
	require.NoError(t, err)
	assert.Empty(t, detached)

	b, err := os.ReadFile(path)
	require.NoError(t, err)

	// Keys are emitted alphabetically; tgNested expands under `first` (its
	// canonical position in declaration order), so `by_name`, which resolves
	// to the same type, carries no `type` or `fields` keys. `plain` has no
	// content and is omitted entirely.
	expected := annotationsFileHeader + `by_name:
  "description": |-
    Map field.
  "markdown_description": |-
    Map field. See [_](/foo.md).
first:
  "description": |-
    First field.
  "type":
    "description": |-
      A nested type.
  "fields":
    "again":
      "description": |-
        Recursive field.
    "description":
      "description": |-
        PLACEHOLDER
items:
  "fields":
    "inner":
      "description": |-
        Inner docs.
mode:
  "type":
    "enum":
      - |-
        a
      - |-
        b
`
	assert.Equal(t, expected, string(b))

	loaded, unknown, err := loadAnnotationsFile(path, g)
	require.NoError(t, err)
	assert.Empty(t, unknown)
	assert.Equal(t, data, loaded)

	// Saving the loaded data again must be byte-identical.
	path2 := filepath.Join(t.TempDir(), "annotations.yml")
	_, err = saveAnnotationsFile(path2, loaded, g)
	require.NoError(t, err)
	b2, err := os.ReadFile(path2)
	require.NoError(t, err)
	assert.Equal(t, string(b), string(b2))
}

func TestAnnotationsFileUnknownEntries(t *testing.T) {
	g, err := newTypeGraph(reflect.TypeFor[tgRoot]())
	require.NoError(t, err)

	path := filepath.Join(t.TempDir(), "annotations.yml")
	err = os.WriteFile(path, []byte(`first:
  description: |-
    Valid entry.
  bogus_key: |-
    Not a descriptor key.
  fields:
    "_":
      description: |-
        The old type docs spelling is not a field.
    no_such_field:
      description: |-
        Field does not exist.
plain:
  fields:
    nested:
      description: |-
        Primitive fields have no nested fields.
  type:
    description: |-
      Primitive fields have no type docs.
`), 0o644)
	require.NoError(t, err)

	data, unknown, err := loadAnnotationsFile(path, g)
	require.NoError(t, err)
	assert.Equal(t, []string{
		"first.bogus_key",
		"first.fields._",
		"first.fields.no_such_field",
		"plain.fields",
		"plain.type",
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

	path := filepath.Join(t.TempDir(), "annotations.yml")
	detached, err := saveAnnotationsFile(path, data, g)
	require.NoError(t, err)
	assert.Equal(t, []string{
		"github.com/databricks/cli/bundle/internal/schema.tgRoot: no_such_field",
		"github.com/databricks/cli/no/such.Type: (type)",
		"github.com/databricks/cli/no/such.Type: field",
	}, detached)
}

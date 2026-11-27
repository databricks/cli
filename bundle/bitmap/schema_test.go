package bitmap

import (
	"bytes"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMergeAppendOnly(t *testing.T) {
	old := []string{"a", "b", "c"}
	// fresh drops "b" (removed field) and adds "d" and "e".
	fresh := []string{"a", "c", "d", "e"}

	merged, added := Merge(old, fresh)

	// Removed fields stay to keep bit positions stable; new ones are appended.
	assert.Equal(t, []string{"a", "b", "c", "d", "e"}, merged)
	assert.Equal(t, []string{"d", "e"}, added)
}

func TestMergeNoChange(t *testing.T) {
	old := []string{"a", "b"}
	merged, added := Merge(old, []string{"a", "b"})
	assert.Equal(t, old, merged)
	assert.Empty(t, added)
}

func TestParseSchema(t *testing.T) {
	assert.Equal(t, []string{"variables", "bundle.name"}, parseSchema([]byte("variables\nbundle.name\n")))

	// Blank lines (leading, internal, trailing) are dropped so they can never
	// shift bit indices in the append-only schema.
	assert.Equal(t, []string{"variables", "bundle.name"}, parseSchema([]byte("\nvariables\n\nbundle.name\n\n")))
	assert.Empty(t, parseSchema([]byte("")))
	assert.Empty(t, parseSchema([]byte("\n\n")))
}

// The databricks-bundles decoder ships its own copy of the schema. A stale copy
// still decodes every bit it has, so only a byte comparison catches it.
func TestPythonSchemaMatchesEmbedded(t *testing.T) {
	python, err := os.ReadFile("../../python/databricks/bundles/_bitmap/schema.deflate")
	require.NoError(t, err)
	assert.True(t, bytes.Equal(schemaDeflate, python), "python/databricks/bundles/_bitmap/schema.deflate differs from bundle/bitmap/schema.deflate; run 'task generate-bitmap-schema'")
}

func TestCheckAppendOnly(t *testing.T) {
	assert.NoError(t, CheckAppendOnly([]string{"a", "b"}, []string{"a", "b"}))
	assert.NoError(t, CheckAppendOnly([]string{"a", "b"}, []string{"a", "b", "c"}))
	assert.NoError(t, CheckAppendOnly(nil, []string{"a"}))
	assert.ErrorContains(t, CheckAppendOnly([]string{"a", "b"}, []string{"b", "a"}), `bit 0 is "a" in the base schema but "b" now`)
	assert.ErrorContains(t, CheckAppendOnly([]string{"a", "b"}, []string{"a", "c", "b"}), `bit 1 is "b"`)
	assert.ErrorContains(t, CheckAppendOnly([]string{"a", "b"}, []string{"a"}), "fewer than the 2")
}

func TestDecompressSchemaRoundTrip(t *testing.T) {
	compressed, err := CompressSchema([]byte("a\nb\n"))
	require.NoError(t, err)
	schema, err := DecompressSchema(compressed)
	require.NoError(t, err)
	assert.Equal(t, []string{"a", "b"}, schema)
}

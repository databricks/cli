package debug

import (
	"reflect"
	"testing"

	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/databricks-sdk-go/service/pipelines"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConvertChangePathToDynPath_WithArrayIndex(t *testing.T) {
	// Test case: path with array index like ["Libraries", "1", "Glob"]
	// This simulates the path from r3labs/diff when comparing pipeline libraries
	path := []string{"Libraries", "1", "Glob"}
	structType := reflect.TypeOf(pipelines.PipelineSpec{})

	dynPath, err := convertChangePathToDynPath(path, structType)
	require.NoError(t, err)

	// Expected path: libraries[1].glob
	// Should have 3 components: Key("libraries"), Index(1), Key("glob")
	assert.Equal(t, 3, len(dynPath))

	// First component should be a key "libraries"
	assert.Equal(t, dyn.Key("libraries"), dynPath[0])

	// Second component should be an index 1
	assert.Equal(t, dyn.Index(1), dynPath[1])

	// Third component should be a key "glob"
	assert.Equal(t, dyn.Key("glob"), dynPath[2])
}

func TestConvertChangePathToDynPath_WithMultipleArrayIndices(t *testing.T) {
	// Test case: path with multiple array indices
	path := []string{"Libraries", "0"}
	structType := reflect.TypeOf(pipelines.PipelineSpec{})

	dynPath, err := convertChangePathToDynPath(path, structType)
	require.NoError(t, err)

	// Expected path: libraries[0]
	assert.Equal(t, 2, len(dynPath))
	assert.Equal(t, dyn.Key("libraries"), dynPath[0])
	assert.Equal(t, dyn.Index(0), dynPath[1])
}

func TestConvertChangePathToDynPath_WithoutArrayIndex(t *testing.T) {
	// Test case: simple path without array indices
	path := []string{"Name"}
	structType := reflect.TypeOf(pipelines.PipelineSpec{})

	dynPath, err := convertChangePathToDynPath(path, structType)
	require.NoError(t, err)

	// Expected path: name
	assert.Equal(t, 1, len(dynPath))
	assert.Equal(t, dyn.Key("name"), dynPath[0])
}

func TestConvertChangePathToDynPath_NestedStructWithArrayIndex(t *testing.T) {
	// Test case: array of structs, accessing a field in array element
	// ["Libraries", "1", "Notebook", "Path"]
	path := []string{"Libraries", "1", "Notebook", "Path"}
	structType := reflect.TypeOf(pipelines.PipelineSpec{})

	dynPath, err := convertChangePathToDynPath(path, structType)
	require.NoError(t, err)

	// Expected: libraries[1].notebook.path
	assert.Equal(t, 4, len(dynPath))
	assert.Equal(t, dyn.Key("libraries"), dynPath[0])
	assert.Equal(t, dyn.Index(1), dynPath[1])
	assert.Equal(t, dyn.Key("notebook"), dynPath[2])
	assert.Equal(t, dyn.Key("path"), dynPath[3])
}

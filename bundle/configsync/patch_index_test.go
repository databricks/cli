package configsync

import (
	"testing"

	"github.com/palantir/pkg/yamlpatch/gopkgv3yamlpatcher"
	"github.com/palantir/pkg/yamlpatch/yamlpatch"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

// TestYamlPatchSequentialIndices tests whether yamlpatch operations
// use indices from the original state or after previous operations.
func TestYamlPatchSequentialIndices(t *testing.T) {
	// Initial YAML with 3 items
	yamlContent := `foo:
  - val: 1
    key: a
  - val: 2
    key: b
  - val: 3
    key: c
`

	// Parse paths for operations
	removePath, err := yamlpatch.ParsePath("/foo/0")
	require.NoError(t, err)

	updatePath, err := yamlpatch.ParsePath("/foo/1")
	require.NoError(t, err)

	patcher := gopkgv3yamlpatcher.New(gopkgv3yamlpatcher.IndentSpaces(2))

	// Apply operations: remove index 0, then update index 1
	patch := yamlpatch.Patch{
		yamlpatch.Operation{
			Type: yamlpatch.OperationRemove,
			Path: removePath,
		},
		yamlpatch.Operation{
			Type: yamlpatch.OperationReplace,
			Path: updatePath,
			Value: map[string]any{
				"val": 3,
				"key": "c_updated",
			},
		},
	}

	result, err := patcher.Apply([]byte(yamlContent), patch)
	require.NoError(t, err)

	// Parse result to check which item was updated
	var parsed struct {
		Foo []struct {
			Val int    `yaml:"val"`
			Key string `yaml:"key"`
		} `yaml:"foo"`
	}
	err = yaml.Unmarshal(result, &parsed)
	require.NoError(t, err)

	// After removing index 0 (val: 1, key: a), we should have:
	// - index 0: val: 2, key: b (originally at index 1)
	// - index 1: val: 3, key: c (originally at index 2)
	//
	// The question is: does the update to index 1 affect:
	// a) The item with key "b" (original index 1, current index 0 after removal)
	// b) The item with key "c" (original index 2, current index 1 after removal)
	//
	// If indices are resolved AFTER the remove operation, then index 1
	// refers to what was originally at index 2 (key: c).

	require.Len(t, parsed.Foo, 2)

	// Check which item was updated
	t.Logf("Result: %+v", parsed.Foo)

	// If yamlpatch applies operations sequentially (standard JSON Patch behavior),
	// then after removing index 0, the update to index 1 should affect
	// what was originally at index 2 (key: c)
	assert.Equal(t, 2, parsed.Foo[0].Val)
	assert.Equal(t, "b", parsed.Foo[0].Key)
	assert.Equal(t, 3, parsed.Foo[1].Val)
	assert.Equal(t, "c_updated", parsed.Foo[1].Key)
}

// TestYamlPatchSequentialIndices_Separate tests the same scenario
// but applies operations separately to confirm the behavior.
func TestYamlPatchSequentialIndices_Separate(t *testing.T) {
	yamlContent := `foo:
  - val: 1
    key: a
  - val: 2
    key: b
  - val: 3
    key: c
`

	patcher := gopkgv3yamlpatcher.New(gopkgv3yamlpatcher.IndentSpaces(2))

	// First operation: remove index 0
	removePath, err := yamlpatch.ParsePath("/foo/0")
	require.NoError(t, err)

	result, err := patcher.Apply([]byte(yamlContent), yamlpatch.Patch{
		yamlpatch.Operation{
			Type: yamlpatch.OperationRemove,
			Path: removePath,
		},
	})
	require.NoError(t, err)

	t.Logf("After remove:\n%s", string(result))

	// Second operation: update index 1
	updatePath, err := yamlpatch.ParsePath("/foo/1")
	require.NoError(t, err)

	result, err = patcher.Apply(result, yamlpatch.Patch{
		yamlpatch.Operation{
			Type: yamlpatch.OperationReplace,
			Path: updatePath,
			Value: map[string]any{
				"val": 3,
				"key": "c_updated",
			},
		},
	})
	require.NoError(t, err)

	t.Logf("After update:\n%s", string(result))

	// Parse result
	var parsed struct {
		Foo []struct {
			Val int    `yaml:"val"`
			Key string `yaml:"key"`
		} `yaml:"foo"`
	}
	err = yaml.Unmarshal(result, &parsed)
	require.NoError(t, err)

	require.Len(t, parsed.Foo, 2)
	assert.Equal(t, 2, parsed.Foo[0].Val)
	assert.Equal(t, "b", parsed.Foo[0].Key)
	assert.Equal(t, 3, parsed.Foo[1].Val)
	assert.Equal(t, "c_updated", parsed.Foo[1].Key)
}

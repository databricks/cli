package configsync

import (
	"testing"

	"github.com/databricks/cli/libs/dyn"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEnsurePathExists(t *testing.T) {
	t.Run("empty path returns original value", func(t *testing.T) {
		v := dyn.V(map[string]dyn.Value{
			"foo": dyn.V("bar"),
		})

		result, err := ensurePathExists(v, dyn.Path{})
		require.NoError(t, err)
		assert.Equal(t, v, result)
	})

	t.Run("single-level path on existing map", func(t *testing.T) {
		v := dyn.V(map[string]dyn.Value{
			"existing": dyn.V("value"),
		})

		path := dyn.Path{dyn.Key("new")}
		result, err := ensurePathExists(v, path)
		require.NoError(t, err)

		// Original key should still exist
		existing, err := dyn.GetByPath(result, dyn.Path{dyn.Key("existing")})
		require.NoError(t, err)
		assert.Equal(t, "value", existing.MustString())
	})

	t.Run("multi-level nested path creates all intermediate nodes", func(t *testing.T) {
		v := dyn.V(map[string]dyn.Value{})

		path := dyn.Path{
			dyn.Key("level1"),
			dyn.Key("level2"),
			dyn.Key("level3"),
		}

		result, err := ensurePathExists(v, path)
		require.NoError(t, err)

		// Check that all intermediate nodes exist
		level1, err := dyn.GetByPath(result, dyn.Path{dyn.Key("level1")})
		require.NoError(t, err)
		assert.Equal(t, dyn.KindMap, level1.Kind())

		level2, err := dyn.GetByPath(result, dyn.Path{dyn.Key("level1"), dyn.Key("level2")})
		require.NoError(t, err)
		assert.Equal(t, dyn.KindMap, level2.Kind())
	})

	t.Run("partially existing path creates only missing nodes", func(t *testing.T) {
		v := dyn.V(map[string]dyn.Value{
			"resources": dyn.V(map[string]dyn.Value{
				"existing": dyn.V("value"),
			}),
		})

		path := dyn.Path{
			dyn.Key("resources"),
			dyn.Key("jobs"),
			dyn.Key("my_job"),
		}

		result, err := ensurePathExists(v, path)
		require.NoError(t, err)

		// Check that existing data is preserved
		existing, err := dyn.GetByPath(result, dyn.Path{dyn.Key("resources"), dyn.Key("existing")})
		require.NoError(t, err)
		assert.Equal(t, "value", existing.MustString())

		// Check that new intermediate node was created
		jobs, err := dyn.GetByPath(result, dyn.Path{dyn.Key("resources"), dyn.Key("jobs")})
		require.NoError(t, err)
		assert.Equal(t, dyn.KindMap, jobs.Kind())
	})

	t.Run("fully existing path is idempotent", func(t *testing.T) {
		v := dyn.V(map[string]dyn.Value{
			"resources": dyn.V(map[string]dyn.Value{
				"jobs": dyn.V(map[string]dyn.Value{
					"my_job": dyn.V(map[string]dyn.Value{
						"name": dyn.V("test"),
					}),
				}),
			}),
		})

		path := dyn.Path{
			dyn.Key("resources"),
			dyn.Key("jobs"),
			dyn.Key("my_job"),
		}

		result, err := ensurePathExists(v, path)
		require.NoError(t, err)

		// Check that existing nested data is preserved
		name, err := dyn.GetByPath(result, dyn.Path{dyn.Key("resources"), dyn.Key("jobs"), dyn.Key("my_job"), dyn.Key("name")})
		require.NoError(t, err)
		assert.Equal(t, "test", name.MustString())
	})

	t.Run("can set value after ensuring path exists", func(t *testing.T) {
		v := dyn.V(map[string]dyn.Value{})

		path := dyn.Path{
			dyn.Key("resources"),
			dyn.Key("jobs"),
			dyn.Key("my_job"),
		}

		// Ensure path exists
		result, err := ensurePathExists(v, path)
		require.NoError(t, err)

		// Now SetByPath should work without errors
		finalValue := dyn.V(map[string]dyn.Value{
			"name": dyn.V("test_job"),
		})

		result, err = dyn.SetByPath(result, path, finalValue)
		require.NoError(t, err)

		// Verify the value was set correctly
		job, err := dyn.GetByPath(result, path)
		require.NoError(t, err)
		jobMap, ok := job.AsMap()
		require.True(t, ok)
		name, exists := jobMap.GetByString("name")
		require.True(t, exists)
		assert.Equal(t, "test_job", name.MustString())
	})

	t.Run("handles deeply nested paths", func(t *testing.T) {
		v := dyn.V(map[string]dyn.Value{})

		path := dyn.Path{
			dyn.Key("a"),
			dyn.Key("b"),
			dyn.Key("c"),
			dyn.Key("d"),
			dyn.Key("e"),
		}

		result, err := ensurePathExists(v, path)
		require.NoError(t, err)

		// Verify all intermediate nodes exist
		intermediate, err := dyn.GetByPath(result, dyn.Path{dyn.Key("a"), dyn.Key("b"), dyn.Key("c"), dyn.Key("d")})
		require.NoError(t, err)
		assert.Equal(t, dyn.KindMap, intermediate.Kind())
	})

	t.Run("handles path with existing sequence", func(t *testing.T) {
		v := dyn.V(map[string]dyn.Value{
			"tasks": dyn.V([]dyn.Value{
				dyn.V(map[string]dyn.Value{
					"name": dyn.V("task1"),
				}),
			}),
		})

		path := dyn.Path{
			dyn.Key("tasks"),
			dyn.Index(0),
			dyn.Key("timeout"),
		}

		result, err := ensurePathExists(v, path)
		require.NoError(t, err)

		// Original sequence should still exist
		tasks, err := dyn.GetByPath(result, dyn.Path{dyn.Key("tasks")})
		require.NoError(t, err)
		assert.Equal(t, dyn.KindSequence, tasks.Kind())
	})

	t.Run("fails when sequence index does not exist", func(t *testing.T) {
		v := dyn.V(map[string]dyn.Value{})

		path := dyn.Path{
			dyn.Key("tasks"),
			dyn.Index(0),
			dyn.Key("timeout"),
		}

		_, err := ensurePathExists(v, path)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "sequence index does not exist")
	})

	t.Run("creates intermediate maps before sequence", func(t *testing.T) {
		v := dyn.V(map[string]dyn.Value{})

		// First ensure the path up to the sequence exists
		pathToSeq := dyn.Path{
			dyn.Key("resources"),
			dyn.Key("jobs"),
		}

		result, err := ensurePathExists(v, pathToSeq)
		require.NoError(t, err)

		// Manually add a sequence
		result, err = dyn.SetByPath(result, pathToSeq, dyn.V([]dyn.Value{
			dyn.V(map[string]dyn.Value{"name": dyn.V("job1")}),
		}))
		require.NoError(t, err)

		fullPath := dyn.Path{
			dyn.Key("resources"),
			dyn.Key("jobs"),
			dyn.Index(0),
			dyn.Key("tasks"),
		}

		result, err = ensurePathExists(result, fullPath)
		require.NoError(t, err)

		job, err := dyn.GetByPath(result, dyn.Path{dyn.Key("resources"), dyn.Key("jobs"), dyn.Index(0)})
		require.NoError(t, err)
		assert.Equal(t, dyn.KindMap, job.Kind())
	})
}

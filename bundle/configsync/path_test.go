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

		existing, err := dyn.GetByPath(result, dyn.Path{dyn.Key("resources"), dyn.Key("existing")})
		require.NoError(t, err)
		assert.Equal(t, "value", existing.MustString())

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

		result, err := ensurePathExists(v, path)
		require.NoError(t, err)

		finalValue := dyn.V(map[string]dyn.Value{
			"name": dyn.V("test_job"),
		})

		result, err = dyn.SetByPath(result, path, finalValue)
		require.NoError(t, err)

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

		tasks, err := dyn.GetByPath(result, dyn.Path{dyn.Key("tasks")})
		require.NoError(t, err)
		assert.Equal(t, dyn.KindSequence, tasks.Kind())
	})

	t.Run("creates sequence when index does not exist", func(t *testing.T) {
		v := dyn.V(map[string]dyn.Value{})

		path := dyn.Path{
			dyn.Key("tasks"),
			dyn.Index(0),
			dyn.Key("timeout"),
		}

		result, err := ensurePathExists(v, path)
		require.NoError(t, err)

		tasks, err := dyn.GetByPath(result, dyn.Path{dyn.Key("tasks")})
		require.NoError(t, err)
		assert.Equal(t, dyn.KindSequence, tasks.Kind())

		seq, _ := tasks.AsSequence()
		assert.Len(t, seq, 1)

		assert.Equal(t, dyn.KindMap, seq[0].Kind())
	})

	t.Run("creates intermediate maps before sequence", func(t *testing.T) {
		v := dyn.V(map[string]dyn.Value{})

		pathToSeq := dyn.Path{
			dyn.Key("resources"),
			dyn.Key("jobs"),
		}

		result, err := ensurePathExists(v, pathToSeq)
		require.NoError(t, err)

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

	t.Run("creates sequence with multiple elements", func(t *testing.T) {
		v := dyn.V(map[string]dyn.Value{})

		path := dyn.Path{
			dyn.Key("items"),
			dyn.Index(5),
			dyn.Key("value"),
		}

		result, err := ensurePathExists(v, path)
		require.NoError(t, err)

		items, err := dyn.GetByPath(result, dyn.Path{dyn.Key("items")})
		require.NoError(t, err)
		assert.Equal(t, dyn.KindSequence, items.Kind())

		seq, _ := items.AsSequence()
		assert.Len(t, seq, 6)

		for i, elem := range seq {
			assert.Equal(t, dyn.KindMap, elem.Kind(), "element %d should be a map", i)
		}
	})

	t.Run("handles nested paths within created sequence elements", func(t *testing.T) {
		v := dyn.V(map[string]dyn.Value{})

		path := dyn.Path{
			dyn.Key("jobs"),
			dyn.Index(0),
			dyn.Key("tasks"),
			dyn.Key("main"),
		}

		result, err := ensurePathExists(v, path)
		require.NoError(t, err)

		tasks, err := dyn.GetByPath(result, dyn.Path{
			dyn.Key("jobs"),
			dyn.Index(0),
			dyn.Key("tasks"),
		})
		require.NoError(t, err)
		assert.Equal(t, dyn.KindMap, tasks.Kind())
	})
}

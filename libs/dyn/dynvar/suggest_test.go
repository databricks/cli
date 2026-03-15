package dynvar

import (
	"testing"

	"github.com/databricks/cli/libs/dyn"
	"github.com/stretchr/testify/assert"
)

func TestSuggestPathSingleKeyTypo(t *testing.T) {
	root := dyn.V(map[string]dyn.Value{
		"bundle": dyn.V(map[string]dyn.Value{
			"name": dyn.V("my-bundle"),
		}),
	})

	suggestion := SuggestPath(root, dyn.MustPathFromString("bndle.name"))
	assert.Equal(t, "bundle.name", suggestion)
}

func TestSuggestPathMultiLevelTypo(t *testing.T) {
	root := dyn.V(map[string]dyn.Value{
		"resources": dyn.V(map[string]dyn.Value{
			"jobs": dyn.V(map[string]dyn.Value{
				"my_job": dyn.V(map[string]dyn.Value{
					"id": dyn.V("123"),
				}),
			}),
		}),
	})

	suggestion := SuggestPath(root, dyn.MustPathFromString("resources.jbs.my_jb.id"))
	assert.Equal(t, "resources.jobs.my_job.id", suggestion)
}

func TestSuggestPathExactMatch(t *testing.T) {
	root := dyn.V(map[string]dyn.Value{
		"var": dyn.V(map[string]dyn.Value{
			"my_var": dyn.V("hello"),
		}),
	})

	suggestion := SuggestPath(root, dyn.MustPathFromString("var.my_var"))
	assert.Equal(t, "var.my_var", suggestion)
}

func TestSuggestPathNoMatch(t *testing.T) {
	root := dyn.V(map[string]dyn.Value{
		"bundle": dyn.V(map[string]dyn.Value{
			"name": dyn.V("my-bundle"),
		}),
	})

	suggestion := SuggestPath(root, dyn.MustPathFromString("zzzzzzz.name"))
	assert.Equal(t, "", suggestion)
}

func TestSuggestPathIntoNonMap(t *testing.T) {
	root := dyn.V(map[string]dyn.Value{
		"name": dyn.V("hello"),
	})

	suggestion := SuggestPath(root, dyn.MustPathFromString("name.child"))
	assert.Equal(t, "", suggestion)
}

func TestSuggestPathIndexPassthrough(t *testing.T) {
	root := dyn.V(map[string]dyn.Value{
		"tasks": dyn.V([]dyn.Value{
			dyn.V(map[string]dyn.Value{
				"name": dyn.V("task1"),
			}),
		}),
	})

	suggestion := SuggestPath(root, dyn.MustPathFromString("tasks[0].nme"))
	assert.Equal(t, "tasks[0].name", suggestion)
}

func TestSuggestPathNested(t *testing.T) {
	root := dyn.V(map[string]dyn.Value{
		"variables": dyn.V(map[string]dyn.Value{
			"my_cluster_id": dyn.V(map[string]dyn.Value{
				"value": dyn.V("abc-123"),
			}),
		}),
	})

	suggestion := SuggestPath(root, dyn.MustPathFromString("variables.my_clster_id.value"))
	assert.Equal(t, "variables.my_cluster_id.value", suggestion)
}

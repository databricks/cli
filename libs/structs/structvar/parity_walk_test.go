package structvar_test

import (
	"testing"

	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/libs/structs/structpath"
	"github.com/databricks/cli/libs/structs/structvar"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWalkVisitsParentsBeforeChildren(t *testing.T) {
	root := config.Root{
		Bundle: config.Bundle{Name: "x"},
		Sync:   config.Sync{Include: []string{"a", "b"}},
	}

	var visited []string
	err := structvar.Walk(root.View(), func(p *structpath.PathNode, v structvar.View) error {
		if s, ok := v.AsString(); ok {
			visited = append(visited, p.String()+"="+s)
		}
		return nil
	})
	require.NoError(t, err)
	assert.Equal(t, []string{"bundle.name=x", "sync.include[0]=a", "sync.include[1]=b"}, visited)
}

func TestWalkVisitsAllValues(t *testing.T) {
	root, diags := config.LoadFromBytes("test.yml", []byte(`
resources:
  jobs:
    j:
      name: n
      max_concurrent_runs: ${var.x}
      tags:
        a: b
`))
	require.NoError(t, diags.Error())

	var visited []string
	err := structvar.Walk(root.View().Lookup(structpath.MustParsePath("resources.jobs.j")), func(p *structpath.PathNode, _ structvar.View) error {
		visited = append(visited, p.String())
		return nil
	})
	require.NoError(t, err)
	assert.Equal(t, []string{"", "max_concurrent_runs", "name", "tags", "tags.a"}, visited)
}

func TestSetReference(t *testing.T) {
	root, diags := config.LoadFromBytes("test.yml", []byte(`
resources:
  jobs:
    j:
      name: n
      max_concurrent_runs: ${var.x}
`))
	require.NoError(t, diags.Error())

	path := structpath.MustParsePath("resources.jobs.j.max_concurrent_runs")
	require.NoError(t, root.SetReference(path, "${var.y}"))
	s, ok := root.View().Lookup(path).AsString()
	require.True(t, ok)
	assert.Equal(t, "${var.y}", s)

	// A string field holds the reference itself.
	name := structpath.MustParsePath("resources.jobs.j.name")
	require.NoError(t, root.SetReference(name, "${var.y}"))
	assert.Equal(t, "${var.y}", root.Resources.Jobs["j"].Name)
}

package dyn_test

import (
	"testing"

	"github.com/databricks/cli/libs/dyn"
	"github.com/stretchr/testify/assert"
)

func TestToStructPath(t *testing.T) {
	assert.Nil(t, dyn.ToStructPath(nil))

	p := dyn.NewPath(dyn.Key("resources"), dyn.Key("jobs"), dyn.Key("a_job"), dyn.Key("tasks"), dyn.Index(1), dyn.Key("task_key"))
	assert.Equal(t, "resources.jobs.a_job.tasks[1].task_key", dyn.ToStructPath(p).String())

	p = dyn.NewPath(dyn.Key("resources"), dyn.Key("jobs"), dyn.Key("${var.env}_job"))
	assert.Equal(t, "resources.jobs['${var.env}_job']", dyn.ToStructPath(p).String())
}

func TestToStructPaths(t *testing.T) {
	paths := dyn.ToStructPaths(
		dyn.MustPathFromString("a.b"),
		dyn.NewPath(dyn.Key("c"), dyn.Index(1)),
	)
	assert.Len(t, paths, 2)
	assert.Equal(t, "a.b", paths[0].String())
	assert.Equal(t, "c[1]", paths[1].String())
	assert.Empty(t, dyn.ToStructPaths())
}

package run

import (
	"testing"

	"github.com/databricks/bricks/bundle"
	"github.com/databricks/bricks/bundle/config"
	"github.com/databricks/bricks/bundle/config/resources"
	"github.com/stretchr/testify/assert"
)

func TestCollectNoResources(t *testing.T) {
	b := &bundle.Bundle{
		Config: config.Root{
			Resources: config.Resources{},
		},
	}

	_, err := Collect(b, []string{"foo"})
	assert.ErrorContains(t, err, "bundle defines no resources")
}

func TestCollectNoArg(t *testing.T) {
	b := &bundle.Bundle{
		Config: config.Root{
			Resources: config.Resources{
				Jobs: map[string]*resources.Job{
					"foo": {},
				},
			},
		},
	}

	out, err := Collect(b, []string{})
	assert.NoError(t, err)
	assert.Len(t, out, 1)
}

func TestCollectNoArgMultipleResources(t *testing.T) {
	b := &bundle.Bundle{
		Config: config.Root{
			Resources: config.Resources{
				Jobs: map[string]*resources.Job{
					"foo": {},
					"bar": {},
				},
			},
		},
	}

	_, err := Collect(b, []string{})
	assert.ErrorContains(t, err, "bundle defines multiple resources")
}

func TestCollectSingleArg(t *testing.T) {
	b := &bundle.Bundle{
		Config: config.Root{
			Resources: config.Resources{
				Jobs: map[string]*resources.Job{
					"foo": {},
				},
			},
		},
	}

	out, err := Collect(b, []string{"foo"})
	assert.NoError(t, err)
	assert.Len(t, out, 1)
}

func TestCollectSingleArgNotFound(t *testing.T) {
	b := &bundle.Bundle{
		Config: config.Root{
			Resources: config.Resources{
				Jobs: map[string]*resources.Job{
					"foo": {},
				},
			},
		},
	}

	_, err := Collect(b, []string{"bar"})
	assert.ErrorContains(t, err, "no such resource: bar")
}

func TestCollectSingleArgAmbiguous(t *testing.T) {
	b := &bundle.Bundle{
		Config: config.Root{
			Resources: config.Resources{
				Jobs: map[string]*resources.Job{
					"key": {},
				},
				Pipelines: map[string]*resources.Pipeline{
					"key": {},
				},
			},
		},
	}

	_, err := Collect(b, []string{"key"})
	assert.ErrorContains(t, err, "ambiguous: ")
}

func TestCollectSingleArgWithType(t *testing.T) {
	b := &bundle.Bundle{
		Config: config.Root{
			Resources: config.Resources{
				Jobs: map[string]*resources.Job{
					"key": {},
				},
			},
		},
	}

	out, err := Collect(b, []string{"jobs.key"})
	assert.NoError(t, err)
	assert.Len(t, out, 1)
}

func TestCollectMultipleArg(t *testing.T) {
	b := &bundle.Bundle{
		Config: config.Root{
			Resources: config.Resources{
				Jobs: map[string]*resources.Job{
					"foo": {},
					"bar": {},
				},
				Pipelines: map[string]*resources.Pipeline{
					"qux": {},
				},
			},
		},
	}

	out, err := Collect(b, []string{"foo", "bar", "qux"})
	assert.NoError(t, err)
	assert.Len(t, out, 3)
}

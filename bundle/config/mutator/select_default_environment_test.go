package mutator_test

import (
	"context"
	"testing"

	"github.com/databricks/bricks/bundle"
	"github.com/databricks/bricks/bundle/config"
	"github.com/databricks/bricks/bundle/config/mutator"
	"github.com/stretchr/testify/assert"
)

func TestSelectDefaultEnvironmentNoEnvironments(t *testing.T) {
	bundle := &bundle.Bundle{
		Config: config.Root{
			Environments: map[string]*config.Environment{},
		},
	}
	_, err := mutator.SelectDefaultEnvironment().Apply(context.Background(), bundle)
	assert.ErrorContains(t, err, "no environments defined")
}

func TestSelectDefaultEnvironmentSingleEnvironments(t *testing.T) {
	bundle := &bundle.Bundle{
		Config: config.Root{
			Environments: map[string]*config.Environment{
				"foo": {},
			},
		},
	}
	ms, err := mutator.SelectDefaultEnvironment().Apply(context.Background(), bundle)
	assert.NoError(t, err)
	assert.Len(t, ms, 1)
	assert.Equal(t, "SelectEnvironment(foo)", ms[0].Name())
}

func TestSelectDefaultEnvironmentNoDefaults(t *testing.T) {
	bundle := &bundle.Bundle{
		Config: config.Root{
			Environments: map[string]*config.Environment{
				"foo": {},
				"bar": {},
				"qux": {},
			},
		},
	}
	_, err := mutator.SelectDefaultEnvironment().Apply(context.Background(), bundle)
	assert.ErrorContains(t, err, "please specify environment")
}

func TestSelectDefaultEnvironmentNoDefaultsWithNil(t *testing.T) {
	bundle := &bundle.Bundle{
		Config: config.Root{
			Environments: map[string]*config.Environment{
				"foo": nil,
				"bar": nil,
			},
		},
	}
	_, err := mutator.SelectDefaultEnvironment().Apply(context.Background(), bundle)
	assert.ErrorContains(t, err, "please specify environment")
}

func TestSelectDefaultEnvironmentMultipleDefaults(t *testing.T) {
	bundle := &bundle.Bundle{
		Config: config.Root{
			Environments: map[string]*config.Environment{
				"foo": {Default: true},
				"bar": {Default: true},
				"qux": {Default: true},
			},
		},
	}
	_, err := mutator.SelectDefaultEnvironment().Apply(context.Background(), bundle)
	assert.ErrorContains(t, err, "multiple environments are marked as default")
}

func TestSelectDefaultEnvironmentSingleDefault(t *testing.T) {
	bundle := &bundle.Bundle{
		Config: config.Root{
			Environments: map[string]*config.Environment{
				"foo": {},
				"bar": {Default: true},
				"qux": {},
			},
		},
	}
	ms, err := mutator.SelectDefaultEnvironment().Apply(context.Background(), bundle)
	assert.NoError(t, err)
	assert.Len(t, ms, 1)
	assert.Equal(t, "SelectEnvironment(bar)", ms[0].Name())
}

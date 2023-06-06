package mutator_test

import (
	"context"
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/mutator"
	"github.com/stretchr/testify/assert"
)

func TestSelectDefaultEnvironmentNoEnvironments(t *testing.T) {
	bundle := &bundle.Bundle{
		Config: config.Root{
			Environments: map[string]*config.Environment{},
		},
	}
	err := mutator.SelectDefaultEnvironment().Apply(context.Background(), bundle)
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
	err := mutator.SelectDefaultEnvironment().Apply(context.Background(), bundle)
	assert.NoError(t, err)
	assert.Equal(t, "foo", bundle.Config.Bundle.Environment)
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
	err := mutator.SelectDefaultEnvironment().Apply(context.Background(), bundle)
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
	err := mutator.SelectDefaultEnvironment().Apply(context.Background(), bundle)
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
	err := mutator.SelectDefaultEnvironment().Apply(context.Background(), bundle)
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
	err := mutator.SelectDefaultEnvironment().Apply(context.Background(), bundle)
	assert.NoError(t, err)
	assert.Equal(t, "bar", bundle.Config.Bundle.Environment)
}

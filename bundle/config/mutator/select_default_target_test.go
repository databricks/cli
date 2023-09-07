package mutator_test

import (
	"context"
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/mutator"
	"github.com/stretchr/testify/assert"
)

func TestSelectDefaultTargetNoTargets(t *testing.T) {
	bundle := &bundle.Bundle{
		Config: config.Root{
			Targets: map[string]*config.Target{},
		},
	}
	err := mutator.SelectDefaultTarget().Apply(context.Background(), bundle)
	assert.ErrorContains(t, err, "no targets defined")
}

func TestSelectDefaultTargetSingleTargets(t *testing.T) {
	bundle := &bundle.Bundle{
		Config: config.Root{
			Targets: map[string]*config.Target{
				"foo": {},
			},
		},
	}
	err := mutator.SelectDefaultTarget().Apply(context.Background(), bundle)
	assert.NoError(t, err)
	assert.Equal(t, "foo", bundle.Config.Bundle.Target)
}

func TestSelectDefaultTargetNoDefaults(t *testing.T) {
	bundle := &bundle.Bundle{
		Config: config.Root{
			Targets: map[string]*config.Target{
				"foo": {},
				"bar": {},
				"qux": {},
			},
		},
	}
	err := mutator.SelectDefaultTarget().Apply(context.Background(), bundle)
	assert.ErrorContains(t, err, "please specify target")
}

func TestSelectDefaultTargetNoDefaultsWithNil(t *testing.T) {
	bundle := &bundle.Bundle{
		Config: config.Root{
			Targets: map[string]*config.Target{
				"foo": nil,
				"bar": nil,
			},
		},
	}
	err := mutator.SelectDefaultTarget().Apply(context.Background(), bundle)
	assert.ErrorContains(t, err, "please specify target")
}

func TestSelectDefaultTargetMultipleDefaults(t *testing.T) {
	bundle := &bundle.Bundle{
		Config: config.Root{
			Targets: map[string]*config.Target{
				"foo": {Default: true},
				"bar": {Default: true},
				"qux": {Default: true},
			},
		},
	}
	err := mutator.SelectDefaultTarget().Apply(context.Background(), bundle)
	assert.ErrorContains(t, err, "multiple targets are marked as default")
}

func TestSelectDefaultTargetSingleDefault(t *testing.T) {
	bundle := &bundle.Bundle{
		Config: config.Root{
			Targets: map[string]*config.Target{
				"foo": {},
				"bar": {Default: true},
				"qux": {},
			},
		},
	}
	err := mutator.SelectDefaultTarget().Apply(context.Background(), bundle)
	assert.NoError(t, err)
	assert.Equal(t, "bar", bundle.Config.Bundle.Target)
}

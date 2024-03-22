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
	b := &bundle.Bundle{
		Config: config.Root{
			Targets: map[string]*config.Target{},
		},
	}
	err := bundle.Apply(context.Background(), b, mutator.SelectDefaultTarget())
	assert.ErrorContains(t, err, "no targets defined")
}

func TestSelectDefaultTargetSingleTargets(t *testing.T) {
	b := &bundle.Bundle{
		Config: config.Root{
			Targets: map[string]*config.Target{
				"foo": {},
			},
		},
	}
	diags := bundle.Apply(context.Background(), b, mutator.SelectDefaultTarget())
	assert.Empty(t, diags)

	assert.Equal(t, "foo", b.Config.Bundle.Target)
}

func TestSelectDefaultTargetNoDefaults(t *testing.T) {
	b := &bundle.Bundle{
		Config: config.Root{
			Targets: map[string]*config.Target{
				"foo": {},
				"bar": {},
				"qux": {},
			},
		},
	}
	err := bundle.Apply(context.Background(), b, mutator.SelectDefaultTarget())
	assert.ErrorContains(t, err, "please specify target")
}

func TestSelectDefaultTargetNoDefaultsWithNil(t *testing.T) {
	b := &bundle.Bundle{
		Config: config.Root{
			Targets: map[string]*config.Target{
				"foo": nil,
				"bar": nil,
			},
		},
	}
	err := bundle.Apply(context.Background(), b, mutator.SelectDefaultTarget())
	assert.ErrorContains(t, err, "please specify target")
}

func TestSelectDefaultTargetMultipleDefaults(t *testing.T) {
	b := &bundle.Bundle{
		Config: config.Root{
			Targets: map[string]*config.Target{
				"foo": {Default: true},
				"bar": {Default: true},
				"qux": {Default: true},
			},
		},
	}
	err := bundle.Apply(context.Background(), b, mutator.SelectDefaultTarget())
	assert.ErrorContains(t, err, "multiple targets are marked as default")
}

func TestSelectDefaultTargetSingleDefault(t *testing.T) {
	b := &bundle.Bundle{
		Config: config.Root{
			Targets: map[string]*config.Target{
				"foo": {},
				"bar": {Default: true},
				"qux": {},
			},
		},
	}
	diags := bundle.Apply(context.Background(), b, mutator.SelectDefaultTarget())
	assert.Empty(t, diags)

	assert.Equal(t, "bar", b.Config.Bundle.Target)
}

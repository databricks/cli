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
	diags := bundle.Apply(context.Background(), b, mutator.SelectDefaultTarget())
	assert.ErrorContains(t, diags.Error(), "no targets defined")
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
	assert.NoError(t, diags.Error())
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
	diags := bundle.Apply(context.Background(), b, mutator.SelectDefaultTarget())
	assert.ErrorContains(t, diags.Error(), "please specify target")
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
	diags := bundle.Apply(context.Background(), b, mutator.SelectDefaultTarget())
	assert.ErrorContains(t, diags.Error(), "please specify target")
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
	diags := bundle.Apply(context.Background(), b, mutator.SelectDefaultTarget())
	assert.ErrorContains(t, diags.Error(), "multiple targets are marked as default")
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
	assert.NoError(t, diags.Error())
	assert.Equal(t, "bar", b.Config.Bundle.Target)
}

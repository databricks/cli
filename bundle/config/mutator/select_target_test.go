package mutator_test

import (
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/mutator"
	"github.com/databricks/cli/bundle/internal/bundletest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSelectTarget(t *testing.T) {
	b := &bundle.Bundle{
		Config: config.Root{
			Workspace: config.Workspace{
				Host: "foo",
			},
			Targets: map[string]*config.Target{
				"default": {
					Workspace: &config.Workspace{
						Host: "bar",
					},
				},
			},
		},
	}
	diags := bundletest.ApplyLoading(t, b, mutator.SelectTarget("default"))
	require.NoError(t, diags.Error())
	assert.Equal(t, "bar", b.Config.Workspace.Host)
}

func TestSelectTargetNotFound(t *testing.T) {
	b := &bundle.Bundle{
		Config: config.Root{
			Targets: map[string]*config.Target{
				"default": {},
			},
		},
	}
	diags := bundletest.ApplyLoading(t, b, mutator.SelectTarget("doesnt-exist"))
	require.Error(t, diags.Error(), "no targets defined")
}

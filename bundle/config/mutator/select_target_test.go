package mutator_test

import (
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/mutator"
	"github.com/databricks/cli/bundle/config/resources"
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
	diags := bundle.Apply(t.Context(), b, mutator.SelectTarget("default"))
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
	diags := bundle.Apply(t.Context(), b, mutator.SelectTarget("doesnt-exist"))
	require.Error(t, diags.Error(), "no targets defined")
}

func TestSelectTargetMergeFailure(t *testing.T) {
	root, diags := config.LoadFromBytes("databricks.yml", []byte(`
permissions:
  - {group_name: developers, level: CAN_VIEW}
targets:
  test:
    permissions: ${var.acl}
`))
	require.Empty(t, diags)
	b := &bundle.Bundle{Config: *root}

	diags = bundle.Apply(t.Context(), b, mutator.SelectTarget("test"))
	require.ErrorContains(t, diags.Error(), "cannot merge sequence with string")
	assert.Nil(t, b.Target)
	assert.Nil(t, b.TargetPermissionsStart)
	assert.Empty(t, b.Config.Bundle.Target)
	assert.Contains(t, b.Config.Targets, "test")
	assert.Equal(t, []resources.Permission{{GroupName: "developers", Level: "CAN_VIEW"}}, b.Config.Permissions)
}

package mutator_test

import (
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/mutator"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultWorkspaceRoot(t *testing.T) {
	b := &bundle.Bundle{
		Config: config.Root{
			Bundle: config.Bundle{
				Name:   "name",
				Target: "environment",
			},
		},
	}
	diags := bundle.Apply(t.Context(), b, mutator.DefineDefaultWorkspaceRoot())
	require.NoError(t, diags.Error())

	assert.Equal(t, "~/.bundle/name/environment", b.Config.Workspace.RootPath)
}

func TestDefaultWorkspaceRootIsNameTargetScoped(t *testing.T) {
	tcases := []struct {
		name     string
		rootPath string
		scoped   bool
	}{
		{"defaulted", "", true},
		{"name and target references", "~/.bundle/${bundle.name}/${bundle.target}", true},
		{"references under another prefix", "/Workspace/Shared/${bundle.name}/${bundle.target}", true},
		{"trailing slash", "~/.bundle/${bundle.name}/${bundle.target}/", true},
		// Already-resolved segments are indistinguishable from a literal path that
		// happens to match, so they do not count.
		{"resolved values", "~/.bundle/name/environment", false},
		{"target only", "~/.bundle/${bundle.target}", false},
		{"name only", "~/.bundle/${bundle.name}", false},
		{"extra segment below", "~/.bundle/${bundle.name}/${bundle.target}/inner", false},
		{"unrelated path", "/Workspace/Shared/some/path", false},
	}

	for _, tc := range tcases {
		t.Run(tc.name, func(t *testing.T) {
			b := &bundle.Bundle{
				Config: config.Root{
					Bundle:    config.Bundle{Name: "name", Target: "environment"},
					Workspace: config.Workspace{RootPath: tc.rootPath},
				},
			}
			diags := bundle.Apply(t.Context(), b, mutator.DefineDefaultWorkspaceRoot())
			require.NoError(t, diags.Error())

			assert.Equal(t, tc.scoped, b.RootPathIsNameTargetScoped)
		})
	}
}

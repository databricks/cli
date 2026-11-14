package mutator

import (
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/variable"
	"github.com/databricks/cli/bundle/internal/bundletest"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/databricks-sdk-go/service/iam"
	"github.com/stretchr/testify/require"
)

func TestPrependWorkspacePrefix(t *testing.T) {
	testCases := []struct {
		path     string
		expected string
	}{
		{
			path:     "/Users/test",
			expected: "/Workspace/Users/test",
		},
		{
			path:     "/Shared/test",
			expected: "/Workspace/Shared/test",
		},
		{
			path:     "/Workspace/Users/test",
			expected: "/Workspace/Users/test",
		},
		{
			path:     "/Workspace",
			expected: "/Workspace",
		},
		{
			path:     "/Volumes/Users/test",
			expected: "/Volumes/Users/test",
		},
		{
			path:     "~/test",
			expected: "~/test",
		},
		{
			path:     "${workspace.file_path}/test",
			expected: "${workspace.file_path}/test",
		},
	}

	for _, tc := range testCases {
		b := &bundle.Bundle{
			Config: config.Root{
				Workspace: config.Workspace{
					RootPath:     tc.path,
					ArtifactPath: tc.path,
					FilePath:     tc.path,
					StatePath:    tc.path,
					ResourcePath: tc.path,
				},
			},
		}

		diags := bundle.Apply(t.Context(), b, PrependWorkspacePrefix())
		require.Empty(t, diags)
		require.Equal(t, tc.expected, b.Config.Workspace.RootPath)
		require.Equal(t, tc.expected, b.Config.Workspace.ArtifactPath)
		require.Equal(t, tc.expected, b.Config.Workspace.FilePath)
		require.Equal(t, tc.expected, b.Config.Workspace.StatePath)
		require.Equal(t, tc.expected, b.Config.Workspace.ResourcePath)
	}
}

func TestPrependWorkspacePrefixPreservesLocations(t *testing.T) {
	b := &bundle.Bundle{
		Config: config.Root{
			Workspace: config.Workspace{
				RootPath: "/Users/test",
			},
		},
	}
	locations := []diag.Location{{File: "databricks.yml", Line: 42, Column: 5}}
	bundletest.SetLocation(b, "workspace.root_path", locations)

	diags := bundle.Apply(t.Context(), b, PrependWorkspacePrefix())
	require.Empty(t, diags)
	require.Equal(t, "/Workspace/Users/test", b.Config.Workspace.RootPath)
	require.Equal(t, locations, b.Config.GetLocations("workspace.root_path"))
}

func TestPrependWorkspaceForDefaultConfig(t *testing.T) {
	b := &bundle.Bundle{
		Config: config.Root{
			Bundle: config.Bundle{
				Name:   "test",
				Target: "dev",
			},
			Workspace: config.Workspace{
				CurrentUser: &config.User{
					User: &iam.User{
						UserName: "jane@doe.com",
					},
				},
			},
		},
	}
	diags := bundle.ApplySeq(t.Context(), b, DefineDefaultWorkspaceRoot(), ExpandWorkspaceRoot(), DefineDefaultWorkspacePaths(), PrependWorkspacePrefix())
	require.Empty(t, diags)
	require.Equal(t, "/Workspace/Users/jane@doe.com/.bundle/test/dev", b.Config.Workspace.RootPath)
	require.Equal(t, "/Workspace/Users/jane@doe.com/.bundle/test/dev/artifacts", b.Config.Workspace.ArtifactPath)
	require.Equal(t, "/Workspace/Users/jane@doe.com/.bundle/test/dev/files", b.Config.Workspace.FilePath)
	require.Equal(t, "/Workspace/Users/jane@doe.com/.bundle/test/dev/state", b.Config.Workspace.StatePath)
	require.Equal(t, "/Workspace/Users/jane@doe.com/.bundle/test/dev/resources", b.Config.Workspace.ResourcePath)
}

func TestPrependWorkspacePrefixAfterVariableResolution(t *testing.T) {
	for _, root := range []string{"/Users/jane@doe.com", "/Shared"} {
		t.Run(root, func(t *testing.T) {
			b := &bundle.Bundle{
				Config: config.Root{
					Bundle: config.Bundle{Name: root},
					Workspace: config.Workspace{
						RootPath: "${bundle.name}/path/to/root",
					},
					Variables: map[string]*variable.Variable{
						"config_file": {Value: "${workspace.file_path}/config.yaml"},
					},
				},
			}

			diags := bundle.ApplySeq(t.Context(), b,
				DefineDefaultWorkspacePaths(),
				PrependWorkspacePrefix(),
				ResolveVariableReferencesInWorkspacePaths(),
			)
			require.NoError(t, diags.Error())
			require.Equal(t, root+"/path/to/root", b.Config.Workspace.RootPath)
			require.Equal(t, "${workspace.file_path}/config.yaml", b.Config.Variables["config_file"].Value)

			diags = bundle.ApplySeq(t.Context(), b, PrependWorkspacePrefix(), ResolveVariableReferencesWithoutResources())
			require.NoError(t, diags.Error())
			require.Equal(t, "/Workspace"+root+"/path/to/root", b.Config.Workspace.RootPath)
			require.Equal(t, "/Workspace"+root+"/path/to/root/files", b.Config.Workspace.FilePath)
			require.Equal(t, "/Workspace"+root+"/path/to/root/artifacts", b.Config.Workspace.ArtifactPath)
			require.Equal(t, "/Workspace"+root+"/path/to/root/state", b.Config.Workspace.StatePath)
			require.Equal(t, "/Workspace"+root+"/path/to/root/resources", b.Config.Workspace.ResourcePath)
			require.Equal(t, "/Workspace"+root+"/path/to/root/files/config.yaml", b.Config.Variables["config_file"].Value)
		})
	}
}

func TestPrependWorkspacePrefixSourceLinkedAlias(t *testing.T) {
	b := &bundle.Bundle{
		SyncRootPath: "/Workspace/Users/jane/source",
		Config: config.Root{
			Workspace: config.Workspace{
				RootPath: "/Users/jane/bundle",
				FilePath: "${workspace.root_path}/files",
				SyncMode: config.SyncModeInPlace,
			},
			Variables: map[string]*variable.Variable{
				"config_file": {Value: "${workspace.file_path}/config.yaml"},
			},
		},
	}

	diags := bundle.ApplySeq(t.Context(), b,
		PrependWorkspacePrefix(),
		ResolveVariableReferencesInWorkspacePaths(),
		PrependWorkspacePrefix(),
		ResolveVariableReferencesWithoutResources(),
	)
	require.NoError(t, diags.Error())
	require.Equal(t, "/Workspace/Users/jane/bundle/files", b.Config.Workspace.FilePath)
	require.Equal(t, "/Workspace/Users/jane/source/config.yaml", b.Config.Variables["config_file"].Value)
}

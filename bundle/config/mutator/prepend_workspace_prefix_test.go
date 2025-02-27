package mutator

import (
	"context"
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
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

		diags := bundle.Apply(context.Background(), b, PrependWorkspacePrefix())
		require.Empty(t, diags)
		require.Equal(t, tc.expected, b.Config.Workspace.RootPath)
		require.Equal(t, tc.expected, b.Config.Workspace.ArtifactPath)
		require.Equal(t, tc.expected, b.Config.Workspace.FilePath)
		require.Equal(t, tc.expected, b.Config.Workspace.StatePath)
		require.Equal(t, tc.expected, b.Config.Workspace.ResourcePath)
	}
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
	diags := bundle.ApplySeq(context.Background(), b, DefineDefaultWorkspaceRoot(), ExpandWorkspaceRoot(), DefineDefaultWorkspacePaths(), PrependWorkspacePrefix())
	require.Empty(t, diags)
	require.Equal(t, "/Workspace/Users/jane@doe.com/.bundle/test/dev", b.Config.Workspace.RootPath)
	require.Equal(t, "/Workspace/Users/jane@doe.com/.bundle/test/dev/artifacts", b.Config.Workspace.ArtifactPath)
	require.Equal(t, "/Workspace/Users/jane@doe.com/.bundle/test/dev/files", b.Config.Workspace.FilePath)
	require.Equal(t, "/Workspace/Users/jane@doe.com/.bundle/test/dev/state", b.Config.Workspace.StatePath)
	require.Equal(t, "/Workspace/Users/jane@doe.com/.bundle/test/dev/resources", b.Config.Workspace.ResourcePath)
}

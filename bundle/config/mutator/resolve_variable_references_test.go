package mutator

import (
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/databricks-sdk-go/service/pipelines"
	"github.com/stretchr/testify/require"
)

func TestResolveVariableReferencesWithSourceLinkedDeployment(t *testing.T) {
	testCases := []struct {
		enabled bool
		assert  func(t *testing.T, b *bundle.Bundle)
	}{
		{
			true,
			func(t *testing.T, b *bundle.Bundle) {
				// Variables that use workspace file path should have SyncRootValue during resolution phase
				require.Equal(t, "sync/root/path", b.Config.Resources.Pipelines["pipeline1"].Configuration["source"])

				// The file path itself should remain the same
				require.Equal(t, "file/path", b.Config.Workspace.FilePath)
			},
		},
		{
			false,
			func(t *testing.T, b *bundle.Bundle) {
				require.Equal(t, "file/path", b.Config.Resources.Pipelines["pipeline1"].Configuration["source"])
				require.Equal(t, "file/path", b.Config.Workspace.FilePath)
			},
		},
	}

	for _, testCase := range testCases {
		b := &bundle.Bundle{
			SyncRootPath: "sync/root/path",
			Config: config.Root{
				Presets: config.Presets{
					SourceLinkedDeployment: &testCase.enabled,
				},
				Workspace: config.Workspace{
					FilePath: "file/path",
				},
				Resources: config.Resources{
					Pipelines: map[string]*resources.Pipeline{
						"pipeline1": {
							CreatePipeline: pipelines.CreatePipeline{
								Configuration: map[string]string{
									"source": "${workspace.file_path}",
								},
							},
						},
					},
				},
			},
		}

		diags := bundle.Apply(t.Context(), b, ResolveVariableReferencesOnlyResources("workspace"))
		require.NoError(t, diags.Error())
		testCase.assert(t, b)
	}
}

func TestResolveVolumePathReferencesOnlyResources(t *testing.T) {
	b := &bundle.Bundle{
		Config: config.Root{
			Resources: config.Resources{
				Volumes: map[string]*resources.Volume{
					"foo": {
						VolumePath: "/Volumes/main/myschema/myvolume",
					},
					"volumePathRef": {},
					"otherRef":      {},
				},
			},
		},
	}

	b.Config.Resources.Volumes["volumePathRef"].Comment = "${resources.volumes.foo.volume_path}"
	b.Config.Resources.Volumes["otherRef"].Comment = "${resources.volumes.foo.name}"

	diags := bundle.Apply(t.Context(), b, ResolveVolumePathReferencesOnlyResources())
	require.NoError(t, diags.Error())
	require.Equal(t, "/Volumes/main/myschema/myvolume", b.Config.Resources.Volumes["volumePathRef"].Comment)
	require.Equal(t, "${resources.volumes.foo.name}", b.Config.Resources.Volumes["otherRef"].Comment)
}

func TestResolveVolumePathReferencesOnlyResources_MissingTarget(t *testing.T) {
	b := &bundle.Bundle{
		Config: config.Root{
			Resources: config.Resources{
				Volumes: map[string]*resources.Volume{
					"foo": {},
				},
			},
		},
	}
	b.Config.Resources.Volumes["foo"].Comment = "${resources.volumes.missing.volume_path}"

	diags := bundle.Apply(t.Context(), b, ResolveVolumePathReferencesOnlyResources())
	require.ErrorContains(t, diags.Error(), "reference does not exist: ${resources.volumes.missing.volume_path}")
}

func TestResolveVolumePathReferencesOnlyResources_UnsetTargetResolvesToEmpty(t *testing.T) {
	// When the target volume exists but its volume_path was never computed (for
	// example because its name is only known at deploy), the field is normalized
	// to an empty string, so the reference resolves to "" rather than erroring.
	b := &bundle.Bundle{
		Config: config.Root{
			Resources: config.Resources{
				Volumes: map[string]*resources.Volume{
					"foo": {},
					"ref": {},
				},
			},
		},
	}
	b.Config.Resources.Volumes["ref"].Comment = "${resources.volumes.foo.volume_path}"

	diags := bundle.Apply(t.Context(), b, ResolveVolumePathReferencesOnlyResources())
	require.NoError(t, diags.Error())
	require.Empty(t, b.Config.Resources.Volumes["ref"].Comment)
}

func TestIsVolumePathReferencePath(t *testing.T) {
	require.True(t, isVolumePathReferencePath(dyn.MustPathFromString("resources.volumes.foo.volume_path")))
	require.False(t, isVolumePathReferencePath(dyn.MustPathFromString("resources.volumes.foo.name")))
	require.False(t, isVolumePathReferencePath(dyn.MustPathFromString("resources.jobs.foo.name")))
}

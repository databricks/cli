package libraries

import (
	"testing"

	"github.com/databricks/cli/libs/sdktypes/jobs/v2"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/internal/testutil"
	"github.com/stretchr/testify/require"
)

func TestValidateEnvironments(t *testing.T) {
	tmpDir := t.TempDir()
	testutil.Touch(t, tmpDir, "wheel.whl")

	b := &bundle.Bundle{
		SyncRootPath: tmpDir,
		Config: config.Root{
			Resources: config.Resources{
				Jobs: map[string]*resources.Job{
					"job": {
						JobSettings: jobs.JobSettings{
							Environments: []jobs.JobEnvironment{
								{
									Spec: &jobs.Environment{
										Dependencies: []string{
											"./wheel.whl",
											"simplejson",
											"/Workspace/Users/foo@bar.test/artifacts/test.whl",
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	diags := bundle.Apply(t.Context(), b, ExpandGlobReferences())
	require.Nil(t, diags)
}

func TestValidateEnvironmentsNoFile(t *testing.T) {
	tmpDir := t.TempDir()

	b := &bundle.Bundle{
		SyncRootPath: tmpDir,
		Config: config.Root{
			Resources: config.Resources{
				Jobs: map[string]*resources.Job{
					"job": {
						JobSettings: jobs.JobSettings{
							Environments: []jobs.JobEnvironment{
								{
									Spec: &jobs.Environment{
										Dependencies: []string{
											"./wheel.whl",
											"simplejson",
											"/Workspace/Users/foo@bar.test/artifacts/test.whl",
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	diags := bundle.Apply(t.Context(), b, ExpandGlobReferences())
	require.Len(t, diags, 1)
	require.Equal(t, "file doesn't exist ./wheel.whl", diags[0].Summary)
}

func TestValidateTaskLibraries(t *testing.T) {
	tmpDir := t.TempDir()
	testutil.Touch(t, tmpDir, "wheel.whl")

	b := &bundle.Bundle{
		SyncRootPath: tmpDir,
		Config: config.Root{
			Resources: config.Resources{
				Jobs: map[string]*resources.Job{
					"job": {
						JobSettings: jobs.JobSettings{
							Tasks: []jobs.TaskSettings{
								{
									Libraries: []jobs.Library{
										{
											Whl: new("./wheel.whl"),
										},
										{
											Whl: new("/Workspace/Users/foo@bar.test/artifacts/test.whl"),
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	diags := bundle.Apply(t.Context(), b, ExpandGlobReferences())
	require.Nil(t, diags)
}

func TestValidateTaskLibrariesNoFile(t *testing.T) {
	tmpDir := t.TempDir()

	b := &bundle.Bundle{
		SyncRootPath: tmpDir,
		Config: config.Root{
			Resources: config.Resources{
				Jobs: map[string]*resources.Job{
					"job": {
						JobSettings: jobs.JobSettings{
							Tasks: []jobs.TaskSettings{
								{
									Libraries: []jobs.Library{
										{
											Whl: new("./wheel.whl"),
										},
										{
											Whl: new("/Workspace/Users/foo@bar.test/artifacts/test.whl"),
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	diags := bundle.Apply(t.Context(), b, ExpandGlobReferences())
	require.Len(t, diags, 1)
	require.Equal(t, "file doesn't exist ./wheel.whl", diags[0].Summary)
}

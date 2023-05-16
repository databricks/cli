package terraform

import (
	"context"
	"os/exec"
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/stretchr/testify/require"
)

func TestInitEnvironmentVariables(t *testing.T) {
	_, err := exec.LookPath("terraform")
	if err != nil {
		t.Skipf("cannot find terraform binary: %s", err)
	}

	bundle := &bundle.Bundle{
		Config: config.Root{
			Path: t.TempDir(),
			Bundle: config.Bundle{
				Environment: "whatever",
				Terraform: &config.Terraform{
					ExecPath: "terraform",
				},
			},
		},
	}

	// Trigger initialization of workspace client.
	// TODO(pietern): create test fixture that initializes a mocked client.
	t.Setenv("DATABRICKS_HOST", "https://x")
	t.Setenv("DATABRICKS_TOKEN", "foobar")
	bundle.WorkspaceClient()

	_, err = Initialize().Apply(context.Background(), bundle)
	require.NoError(t, err)
}

package terraform

import (
	"context"
	"os/exec"
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/stretchr/testify/require"
)

func TestLoadWithNoState(t *testing.T) {
	_, err := exec.LookPath("terraform")
	if err != nil {
		t.Skipf("cannot find terraform binary: %s", err)
	}

	b := &bundle.Bundle{
		Path: t.TempDir(),
		Config: config.Root{
			Bundle: config.Bundle{
				Target: "whatever",
				Terraform: &config.Terraform{
					ExecPath: "terraform",
				},
			},
		},
	}

	t.Setenv("DATABRICKS_HOST", "https://x")
	t.Setenv("DATABRICKS_TOKEN", "foobar")
	b.WorkspaceClient()

	diags := bundle.Apply(context.Background(), b, bundle.Seq(
		Initialize(),
		Load(ErrorOnEmptyState),
	))

	require.ErrorContains(t, diags.Error(), "Did you forget to run 'databricks bundle deploy'")
}

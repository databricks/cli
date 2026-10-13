package client_test

import (
	"os/exec"
	"testing"

	"github.com/databricks/cli/libs/python"
	"github.com/stretchr/testify/require"
)

func TestSSHServerBootstrap(test *testing.T) {
	test.Parallel()

	cmd := exec.CommandContext(test.Context(), python.GetExecutable(), "testdata/ssh_server_bootstrap_test.py")
	output, err := cmd.CombinedOutput()
	require.NoError(test, err, "%s", output)
}

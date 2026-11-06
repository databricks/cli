package sandbox_test

import (
	"testing"

	sandboxcmd "github.com/databricks/cli/cmd/sandbox"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSandboxFsRejectsInvalidRemotePaths(t *testing.T) {
	for _, tc := range []struct {
		input string
		err   string
	}{
		{"my-sandbox:relative", "sandbox path must be absolute: use my-sandbox:/path"},
		{":/tmp/file", "empty sandbox identifier: use SANDBOX:/path"},
	} {
		t.Run(tc.input, func(t *testing.T) {
			ctx, _ := cmdio.NewTestContextWithStdout(t.Context())
			cmd, _, err := sandboxcmd.New().Find([]string{"fs", "cat"})
			require.NoError(t, err)
			cmd.SetContext(ctx)
			assert.EqualError(t, cmd.RunE(cmd, []string{tc.input}), tc.err)
		})
	}
}

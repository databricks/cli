package sandbox_test

import (
	"strings"
	"testing"

	sandboxcmd "github.com/databricks/cli/cmd/sandbox"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSandboxFsRejectsRelativeRemotePaths(t *testing.T) {
	for _, input := range []string{"my-sandbox:relative", ":/tmp/file"} {
		t.Run(input, func(t *testing.T) {
			ctx, _ := cmdio.NewTestContextWithStdout(t.Context())
			cmd, _, err := sandboxcmd.New().Find([]string{"fs", "cat"})
			require.NoError(t, err)
			cmd.SetContext(ctx)
			err = cmd.RunE(cmd, []string{input})
			require.Error(t, err)
			assert.Contains(t, strings.ToLower(err.Error()), "sandbox")
		})
	}
}

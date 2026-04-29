package generate

import (
	"bytes"
	"context"
	"testing"

	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/flags"
	"github.com/databricks/databricks-sdk-go/experimental/mocks"
	"github.com/spf13/cobra"
)

// stripPreRun clears auth-hook PreRunE/PersistentPreRunE recursively. The
// real generate subcommands set root.MustWorkspaceClient as PreRunE; tests
// stand in their own mock workspace client via cmdctx so the real hook
// would just fail on a missing ~/.databrickscfg.
func stripPreRun(cmd *cobra.Command) {
	cmd.PersistentPreRunE = nil
	cmd.PersistentPreRun = nil
	cmd.PreRunE = nil
	cmd.PreRun = nil
	for _, sub := range cmd.Commands() {
		stripPreRun(sub)
	}
}

// runSubcmd executes the named subcommand from the parent generate group
// with cmdctx pre-populated with the mock workspace client. Returns stderr
// (captured cmdio output) and the cobra error. Args are passed verbatim;
// callers include both the subcommand name and any flags.
func runSubcmd(t *testing.T, w *mocks.MockWorkspaceClient, args ...string) (string, error) {
	t.Helper()
	cmd := New()
	stripPreRun(cmd)

	ctx := context.Background()
	var stderr bytes.Buffer
	cmdIO := cmdio.NewIO(ctx, flags.OutputText, nil, &bytes.Buffer{}, &stderr, "", "")
	ctx = cmdio.InContext(ctx, cmdIO)
	ctx = cmdctx.SetWorkspaceClient(ctx, w.WorkspaceClient)
	cmd.SetContext(ctx)

	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs(args)
	err := cmd.Execute()
	return stderr.String(), err
}

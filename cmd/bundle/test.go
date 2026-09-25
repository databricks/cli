package bundle

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/databricks/cli/libs/execv"
	"github.com/spf13/cobra"
)

var (
	bundleTestLookPath = exec.LookPath
	bundleTestExecv    = execv.Execv
)

func newTestCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "test [bundletest-args...]",
		Short: "Run tests for a bundle with bundletest",
		Long: `Run tests for a bundle with the experimental bundletest runner.

All arguments are passed directly to bundletest. Arguments that bundletest does
not consume are forwarded to pytest, including arguments after --.

Examples:
  databricks bundle test --local -v
  databricks bundle test --local --changed --base origin/main
  databricks bundle test --cloud --profile dev --warehouse-id abc123
  databricks bundle test --local -- -k transform_orders`,
		Args:               cobra.ArbitraryArgs,
		DisableFlagParsing: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return executeBundleTest(args)
		},
	}

	return cmd
}

func executeBundleTest(args []string) error {
	runner, err := bundleTestLookPath("bundletest")
	if err != nil {
		return fmt.Errorf("cannot find bundletest on PATH; install it with `uv pip install -e /path/to/databricks-cli/experimental/bundletest`: %w", err)
	}

	argv := make([]string, 1, len(args)+1)
	argv[0] = runner
	argv = append(argv, args...)
	err = bundleTestExecv(execv.Options{
		Args: argv,
		Env:  os.Environ(),
	})
	if err != nil {
		return fmt.Errorf("failed to run bundletest: %w", err)
	}
	return nil
}

package apps

import (
	"context"
	"strings"

	"github.com/databricks/cli/libs/exec"
)

// runCommand executes the given command and returns any error.
func runCommand(ctx context.Context, appPath string, args []string) error {
	e, err := exec.NewCommandExecutor(appPath)
	if err != nil {
		return err
	}
	e.WithInheritOutput()

	// Safe to join args with spaces here since args are passed directly inside PrepareEnvironment() and GetCommand()
	// and don't contain user input.
	cmd, err := e.StartCommand(ctx, strings.Join(args, " "))
	if err != nil {
		return err
	}
	return cmd.Wait()
}

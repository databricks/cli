// Copied from cmd/root/root.go and adapted for pipelines use.
package root

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/internal/build"
	"github.com/databricks/cli/libs/log"
	"github.com/spf13/cobra"
)

// New is copied from cmd/root/root.go and adapted for pipelines use.
func New(ctx context.Context) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "pipelines",
		Short:   "Pipelines CLI",
		Version: build.GetInfo().Version,

		// Cobra prints the usage string to stderr if a command returns an error.
		// This usage string should only be displayed if an invalid combination of flags
		// is specified and not when runtime errors occur (e.g. resource not found).
		// The usage string is include in [flagErrorFunc] for flag errors only.
		SilenceUsage: true,

		// Silence error printing by cobra. Errors are printed through cmdio.
		SilenceErrors: true,
	}

	// Pass the context along through the command during initialization.
	// It will be overwritten when the command is executed.
	cmd.SetContext(ctx)

	// Initialize flags
	logFlags := initLogFlags(cmd)
	outputFlag := initOutputFlag(cmd)
	initProfileFlag(cmd)
	initTargetFlag(cmd)

	// Deprecated flag. Warn if it is specified.
	initProgressLoggerFlag(cmd, logFlags)

	cmd.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
		var err error

		ctx := cmd.Context()

		// Configure command IO
		ctx, err = outputFlag.initializeIO(ctx, cmd)
		if err != nil {
			return err
		}

		// Configure default logger.
		ctx, err = logFlags.initializeContext(ctx)
		if err != nil {
			return err
		}

		logger := log.GetLogger(ctx)
		logger.Info("start",
			slog.String("version", build.GetInfo().Version),
			slog.String("args", strings.Join(os.Args, ", ")))

		// Configure our user agent with the command that's about to be executed.
		ctx = withCommandInUserAgent(ctx, cmd)
		ctx = withCommandExecIdInUserAgent(ctx)
		ctx = withUpstreamInUserAgent(ctx)
		ctx = root.InjectTestPidToUserAgent(ctx)
		cmd.SetContext(ctx)
		return nil
	}

	cmd.SetFlagErrorFunc(flagErrorFunc)
	cmd.SetVersionTemplate("Pipelines CLI v{{.Version}} (based on Databricks CLI v{{.Version}})\n")
	return cmd
}

// Wrap flag errors to include the usage string.
func flagErrorFunc(c *cobra.Command, err error) error {
	return fmt.Errorf("%w\n\n%s", err, c.UsageString())
}

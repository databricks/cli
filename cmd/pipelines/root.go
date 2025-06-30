package pipelines

import (
	"context"
	"log/slog"
	"os"
	"strings"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/internal/build"
	"github.com/databricks/cli/libs/log"
	"github.com/spf13/cobra"
)

// NewRoot is copied from cmd/root/root.go and adapted for pipelines use.
func NewRoot(ctx context.Context) *cobra.Command {
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
	logFlags := root.InitLogFlags(cmd)
	progressLoggerFlag := root.InitProgressLoggerFlag(cmd, logFlags)
	outputFlag := root.InitOutputFlag(cmd)
	root.InitProfileFlag(cmd)
	root.InitTargetFlag(cmd)

	cmd.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()

		// Configure default logger.
		ctx, err := logFlags.InitializeContext(ctx)
		if err != nil {
			return err
		}

		logger := log.GetLogger(ctx)
		logger.Info("start",
			slog.String("version", build.GetInfo().Version),
			slog.String("args", strings.Join(os.Args, ", ")))

		// Configure progress logger
		ctx, err = progressLoggerFlag.InitializeContext(ctx)
		if err != nil {
			return err
		}
		// set context, so that initializeIO can have the current context
		cmd.SetContext(ctx)

		// Configure command IO
		err = outputFlag.InitializeIO(cmd)
		if err != nil {
			return err
		}
		// get the context back
		ctx = cmd.Context()

		// Configure our user agent with the command that's about to be executed.
		ctx = root.WithCommandInUserAgent(ctx, cmd)
		ctx = root.WithCommandExecIdInUserAgent(ctx)
		ctx = root.WithUpstreamInUserAgent(ctx)
		cmd.SetContext(ctx)
		return nil
	}

	cmd.SetFlagErrorFunc(root.FlagErrorFunc)
	return cmd
}

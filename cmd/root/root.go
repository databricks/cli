package root

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/databricks/cli/internal/build"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/log"
	"github.com/spf13/cobra"
	"golang.org/x/exp/slog"
)

// RootCmd represents the base command when called without any subcommands
var RootCmd = &cobra.Command{
	Use:     "databricks",
	Short:   "Databricks CLI",
	Version: build.GetInfo().Version,

	// Cobra prints the usage string to stderr if a command returns an error.
	// This usage string should only be displayed if an invalid combination of flags
	// is specified and not when runtime errors occur (e.g. resource not found).
	// The usage string is include in [flagErrorFunc] for flag errors only.
	SilenceUsage: true,

	// Silence error printing by cobra. Errors are printed through cmdio.
	SilenceErrors: true,

	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()

		// Configure default logger.
		ctx, err := initializeLogger(ctx)
		if err != nil {
			return err
		}

		logger := log.GetLogger(ctx)
		logger.Info("start",
			slog.String("version", build.GetInfo().Version),
			slog.String("args", strings.Join(os.Args, ", ")))

		// Configure progress logger
		ctx, err = initializeProgressLogger(ctx)
		if err != nil {
			return err
		}
		// set context, so that initializeIO can have the current context
		cmd.SetContext(ctx)

		// Configure command IO
		err = initializeIO(cmd)
		if err != nil {
			return err
		}
		// get the context back
		ctx = cmd.Context()

		// Configure our user agent with the command that's about to be executed.
		ctx = withCommandInUserAgent(ctx, cmd)
		ctx = withUpstreamInUserAgent(ctx)
		cmd.SetContext(ctx)
		return nil
	},
}

// Wrap flag errors to include the usage string.
func flagErrorFunc(c *cobra.Command, err error) error {
	return fmt.Errorf("%w\n\n%s", err, c.UsageString())
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute(cmd *cobra.Command) {
	// TODO: deferred panic recovery
	ctx := context.Background()

	// Run the command
	cmd, err := cmd.ExecuteContextC(ctx)
	if err != nil {
		// If cmdio logger initialization succeeds, then this function logs with the
		// initialized cmdio logger, otherwise with the default cmdio logger
		cmdio.LogError(cmd.Context(), err)
	}

	// Log exit status and error
	// We only log if logger initialization succeeded and is stored in command
	// context
	if logger, ok := log.FromContext(cmd.Context()); ok {
		if err == nil {
			logger.Info("completed execution",
				slog.String("exit_code", "0"))
		} else {
			logger.Error("failed execution",
				slog.String("exit_code", "1"),
				slog.String("error", err.Error()))
		}
	}

	if err != nil {
		os.Exit(1)
	}
}

func init() {
	RootCmd.SetFlagErrorFunc(flagErrorFunc)
	RootCmd.SetVersionTemplate("Databricks CLI v{{.Version}}\n")
}

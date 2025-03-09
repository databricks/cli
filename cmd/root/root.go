package root

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"runtime/debug"
	"strings"

	"github.com/databricks/cli/internal/build"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/dbr"
	"github.com/databricks/cli/libs/log"
	"github.com/spf13/cobra"
)

func New(ctx context.Context) *cobra.Command {
	cmd := &cobra.Command{
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
	}

	// Pass the context along through the command during initialization.
	// It will be overwritten when the command is executed.
	cmd.SetContext(ctx)

	// Initialize flags
	logFlags := initLogFlags(cmd)
	progressLoggerFlag := initProgressLoggerFlag(cmd, logFlags)
	outputFlag := InitOutputFlag(cmd)
	initProfileFlag(cmd)
	initEnvironmentFlag(cmd)
	InitTargetFlag(cmd)

	cmd.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()

		// Configure default logger.
		ctx, err := logFlags.initializeContext(ctx)
		if err != nil {
			return err
		}

		logger := log.GetLogger(ctx)
		logger.Info("start",
			slog.String("version", build.GetInfo().Version),
			slog.String("args", strings.Join(os.Args, ", ")))

		// Configure progress logger
		ctx, err = progressLoggerFlag.initializeContext(ctx)
		if err != nil {
			return err
		}
		// set context, so that initializeIO can have the current context
		cmd.SetContext(ctx)

		// Configure command IO
		err = outputFlag.initializeIO(cmd)
		if err != nil {
			return err
		}
		// get the context back
		ctx = cmd.Context()

		// Detect if the CLI is running on DBR and store this on the context.
		ctx = dbr.DetectRuntime(ctx)

		// Configure our user agent with the command that's about to be executed.
		ctx = withCommandInUserAgent(ctx, cmd)
		ctx = withCommandExecIdInUserAgent(ctx)
		ctx = withUpstreamInUserAgent(ctx)
		cmd.SetContext(ctx)
		return nil
	}

	cmd.SetFlagErrorFunc(flagErrorFunc)
	cmd.SetVersionTemplate("Databricks CLI v{{.Version}}\n")
	return cmd
}

// Wrap flag errors to include the usage string.
func flagErrorFunc(c *cobra.Command, err error) error {
	return fmt.Errorf("%w\n\n%s", err, c.UsageString())
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute(ctx context.Context, cmd *cobra.Command) (err error) {
	defer func() {
		r := recover()

		// No panic. Return normally.
		if r == nil {
			return
		}

		version := build.GetInfo().Version
		trace := debug.Stack()

		// Set the error so that the CLI exits with a non-zero exit code.
		err = fmt.Errorf("panic: %v", r)

		fmt.Fprintf(cmd.ErrOrStderr(), `The Databricks CLI unexpectedly had a fatal error.
Please report this issue to Databricks in the form of a GitHub issue at:
https://github.com/databricks/cli

CLI Version: %s

Panic Payload: %v

Stack Trace:
%s`, version, r, string(trace))
	}()

	// Run the command
	cmd, err = cmd.ExecuteContextC(ctx)
	if err != nil && !errors.Is(err, ErrAlreadyPrinted) {
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
		} else if errors.Is(err, ErrAlreadyPrinted) {
			logger.Debug("failed execution",
				slog.String("exit_code", "1"),
			)
		} else {
			logger.Info("failed execution",
				slog.String("exit_code", "1"),
				slog.String("error", err.Error()),
			)
		}
	}

	return err
}

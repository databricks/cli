package root

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/databricks/cli/internal/build"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/dbr"
	"github.com/databricks/cli/libs/env"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/cli/libs/telemetry"
	"github.com/databricks/cli/libs/telemetry/events"
	"github.com/databricks/databricks-sdk-go/client"
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
	outputFlag := initOutputFlag(cmd)
	initProfileFlag(cmd)
	initEnvironmentFlag(cmd)
	initTargetFlag(cmd)

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
func Execute(ctx context.Context, cmd *cobra.Command) error {
	// TODO: Make the logger work here, with the start time, end time and the workspace
	// authentication configuration.

	start := time.Now()
	ctx = telemetry.WithDefaultLogger(ctx)

	// Run the command
	cmd, err := cmd.ExecuteContextC(ctx)
	if err != nil && !errors.Is(err, ErrAlreadyPrinted) {
		// If cmdio logger initialization succeeds, then this function logs with the
		// initialized cmdio logger, otherwise with the default cmdio logger
		cmdio.LogError(cmd.Context(), err)
	}

	end := time.Now()
	exitCode := 0
	if err != nil {
		exitCode = 1
	}

	executionContext := events.ExecutionContext{
		OperatingSystem:    runtime.GOOS,
		DbrVersion:         env.Get(ctx, dbr.EnvVarName),
		FromWebTerminal:    isWebTerminal(ctx),
		CommandExecutionId: cmdExecId,
		ExecutionTimeMs:    end.Sub(start).Milliseconds(),
		ExitCode:           int64(exitCode),
	}

	// TODO: Load the API client directly from the context since we expect
	// it to be set for all CLI commands.
	apiClient, err := client.New(ConfigUsed(ctx))
	if err != nil {
		// TODO remove.
		// Do nothing
	}
	telemetry.Flush(ctx, &executionContext, apiClient)

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

	return err
}

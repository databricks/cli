package root

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"runtime"
	"runtime/debug"
	"strings"
	"time"

	"github.com/databricks/cli/internal/build"
	"github.com/databricks/cli/libs/auth"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/daemon"
	"github.com/databricks/cli/libs/dbr"
	"github.com/databricks/cli/libs/env"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/cli/libs/telemetry"
	"github.com/databricks/cli/libs/telemetry/protos"
	"github.com/databricks/databricks-sdk-go/logger"
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

	ctx = telemetry.WithNewLogger(ctx)
	ctx = dbr.DetectRuntime(ctx)
	startTime := time.Now()

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
	exitCode := 0
	if err != nil {
		exitCode = 1
	}

	uploadTelemetry(cmd.Context(), commandString(cmd), startTime, exitCode)
	return err
}

func uploadTelemetry(ctx context.Context, cmdStr string, startTime time.Time, exitCode int) {
	// Nothing to upload.
	if !telemetry.HasLogs(ctx) {
		return
	}

	// Telemetry is disabled. We don't upload logs.
	if os.Getenv(telemetry.DisableEnvVar) != "" {
		return
	}

	telemetry.SetExecutionContext(ctx, protos.ExecutionContext{
		CmdExecID:       cmdExecId,
		Version:         build.GetInfo().Version,
		Command:         cmdStr,
		OperatingSystem: runtime.GOOS,
		DbrVersion:      env.Get(ctx, dbr.EnvVarName),
		ExecutionTimeMs: time.Since(startTime).Milliseconds(),
		ExitCode:        int64(exitCode),
	})

	logs := telemetry.GetLogs(ctx)

	in := telemetry.UploadConfig{
		Logs: logs,
	}

	// Default to warn log level. If debug is enabled in the parent process, we set
	// the log level to debug for the telemetry worker as well.
	logLevel := "warn"
	if log.GetLogger(ctx).Enabled(ctx, logger.LevelDebug) {
		logLevel = "debug"
	}

	d := daemon.Daemon{
		Args:        []string{"telemetry", "upload", fmt.Sprintf("--log-level=%s", logLevel)},
		Env:         auth.ProcessEnv(ConfigUsed(ctx)),
		PidFilePath: os.Getenv(telemetry.PidFileEnvVar),
		LogFile:     os.Getenv(telemetry.UploadLogsFileEnvVar),
	}

	err := d.Start()
	if err != nil {
		log.Warnf(ctx, "failed to start telemetry worker: %s", err)
		return
	}

	// If the telemetry worker is started successfully, we write the logs to its stdin.
	b, err := json.Marshal(in)
	if err != nil {
		log.Warnf(ctx, "failed to marshal telemetry logs: %s", err)
		return
	}
	err = d.WriteInput(b)
	if err != nil {
		log.Warnf(ctx, "failed to write to telemetry worker: %s", err)
	}

	err = d.Release()
	if err != nil {
		log.Warnf(ctx, "failed to release telemetry worker: %s", err)
	}
}

package root

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/databricks/cli/internal/build"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/dbr"
	"github.com/databricks/cli/libs/env"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/cli/libs/telemetry"
	"github.com/databricks/cli/libs/telemetry/protos"
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

// TODO CONTINUE: This setup should mostly work. There are a couple of open questions:
// 4. I can print the output from the telemetry-worker command and a waiting mode
//    to the root.Execution method here to see whether the expected output matches.

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.

// TODO: The test runner also relies on this function. Create a separate function to
// avoid logging telemetry in our testcli runner.
func Execute(ctx context.Context, cmd *cobra.Command) error {
	ctx = telemetry.WithNewLogger(ctx)
	ctx = dbr.DetectRuntime(ctx)
	start := time.Now()

	// Run the command
	cmd, cmdErr := cmd.ExecuteContextC(ctx)
	if cmdErr != nil && !errors.Is(cmdErr, ErrAlreadyPrinted) {
		// If cmdio logger initialization succeeds, then this function logs with the
		// initialized cmdio logger, otherwise with the default cmdio logger
		cmdio.LogError(cmd.Context(), cmdErr)
	}

	// Log exit status and error
	// We only log if logger initialization succeeded and is stored in command
	// context
	if logger, ok := log.FromContext(cmd.Context()); ok {
		if cmdErr == nil {
			logger.Info("completed execution",
				slog.String("exit_code", "0"))
		} else {
			logger.Error("failed execution",
				slog.String("exit_code", "1"),
				slog.String("error", cmdErr.Error()))
		}
	}

	end := time.Now()

	exitCode := 0
	if cmdErr != nil {
		exitCode = 1
	}

	if env.Get(ctx, telemetry.SkipEnvVar) != "true" {
		logTelemetry(ctx, commandString(cmd), start, end, exitCode)
	}

	return cmdErr
}

// TODO: Do not log for integration tests using the CLI.
// TODO: Skip telemetry if the credentials are invalid.
func logTelemetry(ctx context.Context, cmdStr string, start, end time.Time, exitCode int) {
	telemetry.SetExecutionContext(ctx, protos.ExecutionContext{
		CmdExecID:       cmdExecId,
		Version:         build.GetInfo().Version,
		Command:         cmdStr,
		OperatingSystem: runtime.GOOS,
		DbrVersion:      env.Get(ctx, dbr.EnvVarName),
		FromWebTerminal: isWebTerminal(ctx),
		ExecutionTimeMs: end.Sub(start).Milliseconds(),
		ExitCode:        int64(exitCode),
	})

	// TODO: Better check?
	// Do not log telemetry for the telemetry-worker command to avoid fork bombs.
	if cmdStr == "telemetry-worker" {
		return
	}

	execPath, err := os.Executable()
	if err != nil {
		log.Debugf(ctx, "failed to get executable path: %s", err)
	}
	telemetryCmd := exec.Command(execPath, "telemetry-worker")

	// TODO: Add test that ensures that  the context key for cli commands stores a
	// resolved auth configuration.
	// TODO: Add test that the worker inherits the environment variables from the
	// parent process.
	in := telemetry.WorkerInput{
		AuthConfig: ConfigUsed(ctx),
		Logs:       telemetry.GetLogs(ctx),
	}

	if len(in.Logs) == 0 {
		return
	}

	b, err := json.Marshal(in)
	if err != nil {
		log.Debugf(ctx, "failed to marshal telemetry logs: %s", err)
		return
	}

	stdin, err := telemetryCmd.StdinPipe()
	if err != nil {
		log.Debugf(ctx, "failed to create stdin pipe for telemetry worker: %s", err)
	}

	stdout, err := telemetryCmd.StdoutPipe()
	if err != nil {
		log.Debugf(ctx, "failed to create stdout pipe for telemetry worker: %s", err)
	}

	err = telemetryCmd.Start()
	if err != nil {
		log.Debugf(ctx, "failed to start telemetry worker: %s", err)
		return
	}

	// Set DATABRICKS_CLI_SKIP_TELEMETRY to true to ensure that the telemetry worker
	// command accidentally does not call itself causing a fork bomb. This can happen
	// if a change starts logging telemetry in the telemetry worker command's code
	// path.
	telemetryCmd.Env = os.Environ()
	telemetryCmd.Env = append(telemetryCmd.Env, telemetry.SkipEnvVar+"=true")

	_, err = stdin.Write(b)
	if err != nil {
		log.Debugf(ctx, "failed to write to telemetry worker: %s", err)
	}

	err = stdin.Close()
	if err != nil {
		log.Debugf(ctx, "failed to close stdin for telemetry worker: %s", err)
	}

	// This is only meant for testing purposes, to do assertions on the output
	// of the telemetry worker command.
	if env.Get(ctx, telemetry.BlockOnUploadEnvVar) == "true" {
		err = telemetryCmd.Wait()
		if err != nil {
			log.Debugf(ctx, "failed to wait for telemetry worker: %s", err)
		}

		cmdio.LogString(ctx, "telemetry-worker output:")
		b, err := io.ReadAll(stdout)
		if err != nil {
			log.Debugf(ctx, "failed to read telemetry worker output: %s", err)
		}
		cmdio.LogString(ctx, string(b))
	}
}

package root

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"runtime"
	"slices"
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
func Execute(ctx context.Context, cmd *cobra.Command) error {
	// TODO: deferred panic recovery
	ctx = telemetry.WithNewLogger(ctx)
	ctx = dbr.DetectRuntime(ctx)
	start := time.Now()

	// Run the command
	cmd, err := cmd.ExecuteContextC(ctx)
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

	end := time.Now()
	exitCode := 0
	if err != nil {
		exitCode = 1
	}

	uploadTelemetry(cmd.Context(), commandString(cmd), start, end, exitCode)
	return err
}

// We want child telemetry processes to inherit environment variables like $HOME or $HTTPS_PROXY
// because they influence auth resolution.
func inheritEnvVars() []string {
	base := os.Environ()
	out := []string{}
	authEnvVars := auth.EnvVars()

	// Remove any existing auth environment variables. This is done because
	// the CLI offers multiple modalities of configuring authentication like
	// `--profile` or `DATABRICKS_CONFIG_PROFILE` or `profile: <profile>` in the
	// bundle config file.
	//
	// Each of these modalities have different priorities and thus we don't want
	// any auth configuration to piggyback into the child process environment.
	//
	// This is a precaution to avoid conflicting auth configurations being passed
	// to the child telemetry process.
	for _, v := range base {
		k, _, found := strings.Cut(v, "=")
		if !found {
			continue
		}
		if slices.Contains(authEnvVars, k) {
			continue
		}
		out = append(out, v)
	}

	return out
}

// TODO: Add tests validating the auth resolution in the telemetry worker.
func uploadTelemetry(ctx context.Context, cmdStr string, start, end time.Time, exitCode int) {
	// Nothing to upload.
	if !telemetry.HasLogs(ctx) {
		return
	}

	// Telemetry is disabled. We don't upload logs.
	if _, ok := os.LookupEnv(telemetry.DisableEnvVar); ok {
		return
	}

	telemetry.SetExecutionContext(ctx, protos.ExecutionContext{
		CmdExecID:       cmdExecId,
		Version:         build.GetInfo().Version,
		Command:         cmdStr,
		OperatingSystem: runtime.GOOS,
		DbrVersion:      env.Get(ctx, dbr.EnvVarName),
		ExecutionTimeMs: end.Sub(start).Milliseconds(),
		ExitCode:        int64(exitCode),
	})

	logs := telemetry.GetLogs(ctx)

	in := telemetry.UploadConfig{
		Logs: logs,
	}

	// Compute environment variables with the appropriate auth configuration.
	env := inheritEnvVars()
	for k, v := range auth.Env(ConfigUsed(ctx)) {
		env = append(env, fmt.Sprintf("%s=%s", k, v))
	}

	d := daemon.Daemon{
		Args:        []string{"telemetry", "upload"},
		Env:         env,
		PidFilePath: os.Getenv(telemetry.PidFileEnvVar),
		LogFile:     os.Getenv(telemetry.UploadLogsFileEnvVar),
	}

	err := d.Start()
	if err != nil {
		log.Debugf(ctx, "failed to start telemetry worker: %s", err)
		return
	}

	// If the telemetry worker is started successfully, we write the logs to its stdin.
	b, err := json.Marshal(in)
	if err != nil {
		log.Debugf(ctx, "failed to marshal telemetry logs: %s", err)
		return
	}
	err = d.WriteInput(b)
	if err != nil {
		log.Debugf(ctx, "failed to write to telemetry worker: %s", err)
	}

	err = d.Release()
	if err != nil {
		log.Debugf(ctx, "failed to release telemetry worker: %s", err)
	}
}

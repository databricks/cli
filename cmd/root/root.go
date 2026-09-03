package root

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"runtime"
	"runtime/debug"
	"strings"
	"time"

	"github.com/databricks/cli/internal/build"
	"github.com/databricks/cli/libs/aitools/agents"
	"github.com/databricks/cli/libs/aitools/installer"
	"github.com/databricks/cli/libs/auth"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/dbr"
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
	outputFlag := initOutputFlag(cmd)
	initProfileFlag(cmd)
	initEnvironmentFlag(cmd)
	initTargetFlag(cmd)

	// Deprecated flag. Warn if it is specified.
	initProgressLoggerFlag(cmd, logFlags)

	cmd.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
		var err error

		ctx := cmd.Context()
		if verboseCommand := verboseErrorTipCommand(cmd); verboseCommand != nil {
			logFlags.debug = logFlags.debug || cmd.Flag("verbose").Value.String() == "true"
		}

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
			slog.String("args", strings.Join(commandArgsForLogging(cmd, os.Args), ", ")))

		// Configure our user agent with the command that's about to be executed.
		ctx = withCommandInUserAgent(ctx, cmd)
		ctx = withCommandExecIdInUserAgent(ctx)
		ctx = withUpstreamInUserAgent(ctx)
		ctx = withInteractiveModeInUserAgent(ctx)
		ctx = installer.WithAiToolsInUserAgent(ctx)
		ctx = installer.WithAiDevKitInUserAgent(ctx)
		ctx = InjectTestPidToUserAgent(ctx)
		cmd.SetContext(ctx)

		// Recommend installing Databricks AI tooling to Claude Code when it is
		// driving the CLI without the tooling installed (best-effort, stderr only).
		agents.MaybeHint(ctx, cmd)
		return nil
	}

	cmd.PersistentPostRunE = func(cmd *cobra.Command, args []string) error {
		// Wait for any active Bubble Tea programs to finish and restore terminal state
		cmdio.Wait(cmd.Context())
		return nil
	}

	cmd.SetFlagErrorFunc(flagErrorFunc)
	cmd.SetVersionTemplate("Databricks CLI v{{.Version}}\n")
	return cmd
}

// flagErrorFunc wraps flag errors to include the usage string and, for unknown
// flags, a "Did you mean" suggestion based on Levenshtein distance.
func flagErrorFunc(c *cobra.Command, err error) error {
	err = suggestFlagFromError(c, err)
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

	// Configure a telemetry logger and store it in the context.
	ctx = telemetry.WithNewLogger(ctx)

	// Detect if the CLI is running on DBR and store this on the context.
	ctx = dbr.DetectRuntime(ctx)

	// Set a command execution ID value in the context
	ctx = cmdctx.GenerateExecId(ctx)

	startTime := time.Now()

	// Run the command
	cmd, err = cmd.ExecuteContextC(ctx)
	if err != nil && !errors.Is(err, ErrAlreadyPrinted) {
		if cmdctx.HasConfigUsed(cmd.Context()) {
			cfg := cmdctx.ConfigUsed(cmd.Context())
			err = auth.EnrichAuthError(cmd.Context(), cfg, err)
		}
		// A workspace client on the context means the command operates against
		// a workspace; see AppendAccountHostHint for why every error from such
		// commands gets the account-console-host note.
		if cmdctx.HasWorkspaceClient(cmd.Context()) {
			err = auth.AppendAccountHostHint(cmdctx.WorkspaceClient(cmd.Context()).Config, err)
		}
		fmt.Fprintf(cmd.ErrOrStderr(), "Error: %s\n", err.Error())
		printVerboseErrorTip(cmd)
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

	commandStr := commandString(cmd)
	ctx = cmd.Context()

	telemetryErr := telemetry.Upload(cmd.Context(), protos.ExecutionContext{
		CmdExecID:       cmdctx.ExecId(ctx),
		Version:         build.GetInfo().Version,
		Command:         commandStr,
		OperatingSystem: runtime.GOOS,
		DbrVersion:      dbr.RuntimeVersion(ctx).String(),
		ExecutionTimeMs: time.Since(startTime).Milliseconds(),
		ExitCode:        int64(exitCode),
	})
	if telemetryErr != nil {
		log.Infof(ctx, "telemetry upload failed: %s", telemetryErr)
	}

	return err
}

const verboseErrorTipAnnotation = "databricks.cli.verbose-error-tip"

// EnableVerboseErrorTip adds an AIR-scoped verbose flag and opts the command
// into rerun guidance when one of its descendants fails.
func EnableVerboseErrorTip(cmd *cobra.Command) {
	if cmd.Annotations == nil {
		cmd.Annotations = make(map[string]string)
	}
	cmd.Annotations[verboseErrorTipAnnotation] = "true"
	cmd.PersistentFlags().BoolP("verbose", "v", false, "enable debug logging")
}

func verboseErrorTipCommand(cmd *cobra.Command) *cobra.Command {
	for current := cmd; current != nil; current = current.Parent() {
		if current.Annotations[verboseErrorTipAnnotation] == "true" {
			return current
		}
	}
	return nil
}

func commandArgsForLogging(cmd *cobra.Command, args []string) []string {
	if verboseErrorTipCommand(cmd) == nil {
		return args
	}

	redacted := append([]string(nil), args...)
	for i := 0; i < len(redacted); i++ {
		switch {
		case redacted[i] == "--override" && i+1 < len(redacted):
			redacted[i+1] = "<redacted>"
			i++
		case strings.HasPrefix(redacted[i], "--override="):
			redacted[i] = "--override=<redacted>"
		}
	}
	return redacted
}

func printVerboseErrorTip(cmd *cobra.Command) {
	verboseCommand := verboseErrorTipCommand(cmd)
	if verboseCommand == nil {
		return
	}

	debugFlag := cmd.Root().PersistentFlags().Lookup("debug")
	verboseFlag := cmd.Flag("verbose")
	if debugFlag.Value.String() == "true" || verboseFlag.Value.String() == "true" {
		return
	}

	commandPrefix := verboseCommand.CommandPath()
	command := commandPrefix + " -v" + strings.TrimPrefix(cmd.CommandPath(), commandPrefix)
	fmt.Fprintf(
		cmd.ErrOrStderr(),
		"\nTip: use the -v (verbose) flag immediately after air to see more details and a trace of this error:\n  %s …\n",
		command,
	)
}

// This function is used to report an unknown subcommand.
// It is used in the [cobra.Command.RunE] field of commands that have subcommands.
// If user provided a valid subcommand, RunE for the
// If there are any arguments, it means the user has provided an unknown subcommand.
// If there are no arguments, it means the user has not provided any subcommand, and the help
// command should be displayed.
func ReportUnknownSubcommand(cmd *cobra.Command, args []string) error {
	if len(args) > 0 {
		return &InvalidArgsError{
			Message: fmt.Sprintf("unknown command %q for %q", args[0], cmd.CommandPath()),
			Command: cmd,
		}
	}
	return cmd.Help()
}

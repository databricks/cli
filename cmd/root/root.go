package root

import (
	"context"
	"errors"
	"fmt"
	stdlog "log" //nolint:depguard // slog.SetDefault redirects the standard log package, and Execute restores it
	"log/slog"
	"os"
	"runtime"
	"runtime/debug"
	"strconv"
	"strings"
	"time"

	"github.com/databricks/cli/internal/build"
	"github.com/databricks/cli/libs/aitools/agents"
	"github.com/databricks/cli/libs/aitools/autoinstall"
	"github.com/databricks/cli/libs/aitools/installer"
	"github.com/databricks/cli/libs/auth"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/dbr"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/cli/libs/process"
	"github.com/databricks/cli/libs/skills"
	"github.com/databricks/cli/libs/telemetry"
	"github.com/databricks/cli/libs/telemetry/protos"
	"github.com/databricks/cli/libs/textutil"
	"github.com/spf13/cobra"
)

func newRootCommand(ctx context.Context) *cobra.Command {
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
	initSkillsInvokedFlag(cmd)

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
		ctx, err = logFlags.initializeContext(ctx, cmd)
		if err != nil {
			return err
		}

		logger := log.GetLogger(ctx)
		logger.Info("start",
			slog.String("version", build.GetInfo().Version),
			slog.String("args", strings.Join(os.Args, ", ")))

		warnIfStaleVersion(ctx, cmd)

		// Configure our user agent with the command that's about to be executed.
		ctx = withCommandInUserAgent(ctx, cmd)
		ctx = withCommandExecIdInUserAgent(ctx)
		ctx = withUpstreamInUserAgent(ctx)
		ctx = withInteractiveModeInUserAgent(ctx)
		ctx = installer.WithAiToolsInUserAgent(ctx)
		ctx = installer.WithAiDevKitInUserAgent(ctx)
		ctx = installer.WithCallingAgentToolingInUserAgent(ctx)
		ctx = InjectTestPidToUserAgent(ctx)
		ctx = withSkillsInUserAgent(ctx, cmd)
		cmd.SetContext(ctx)

		// Recommend installing Databricks AI tooling to the calling coding agent
		// when it drives the CLI without the tooling installed (best-effort: a
		// stderr line and a telemetry event that is uploaded at exit). Only the
		// global install state is read, as for the calling-agent User-Agent.
		deferred := agents.MaybeHint(ctx, cmd, func(ctx context.Context, agentName string) bool {
			return installer.PluginRecorded(ctx, installer.ScopeGlobal, agentName)
		})
		// A hint withheld for auto-install prints once the command has ended.
		// Commands run without execute have no slot, and then it is dropped.
		if slot, ok := ctx.Value(deferredHintKey{}).(*func()); ok {
			*slot = deferred
		}
		return nil
	}

	cmd.PersistentPostRunE = func(cmd *cobra.Command, args []string) error {
		// Wait for any active Bubble Tea programs to finish and restore terminal state
		cmdio.Wait(cmd.Context())

		// On the first run of a new CLI version, install Databricks AI tools.
		// Cobra runs post-run hooks only when the command succeeded.
		if autoinstall.Run != nil {
			sig := autoinstall.Run(cmd)
			// Commands run without execute have no slot, and then the CLI does
			// not end by the signal.
			if caught, ok := cmd.Context().Value(caughtSignalKey{}).(*os.Signal); ok {
				*caught = sig
			}
		}
		return nil
	}

	cmd.SetFlagErrorFunc(flagErrorFunc)
	cmd.SetVersionTemplate("Databricks CLI v{{.Version}}\n")
	setHelpFunc(cmd)
	return cmd
}

// caughtSignalKey is the context key of the *os.Signal that execute puts in
// the context. It holds the signal that stopped auto-install, if one did.
type caughtSignalKey struct{}

// deferredHintKey is the context key of the *func() that execute puts in the
// context. It holds the agent hint withheld for auto-install, if one was (see
// agents.MaybeHint).
type deferredHintKey struct{}

// raise is process.Raise; tests replace it.
var raise = process.Raise

// flagErrorFunc wraps flag errors to include the usage string and, for unknown
// flags, a "Did you mean" suggestion based on Levenshtein distance.
func flagErrorFunc(c *cobra.Command, err error) error {
	err = suggestFlagFromError(c, err)
	return fmt.Errorf("%w\n\n%s", err, c.UsageString())
}

// execute runs cmd with args, which select the subcommand to run. Unlike
// [cobra.Command.SetArgs], nil args mean no arguments rather than the process
// arguments.
func execute(ctx context.Context, cmd *cobra.Command, args []string) (err error) {
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

	// Cobra runs with the process arguments if args are nil.
	if args == nil {
		args = []string{}
	}
	cmd.SetArgs(args)

	startTime := time.Now()

	// initializeContext puts the log file it opens in this slot. It is closed when Execute returns, even on a panic, and
	// the loggers are restored, so nothing writes to the closed file.
	var logFile logFileSlot
	ctx = context.WithValue(ctx, logFileKey{}, &logFile)
	var caught os.Signal
	ctx = context.WithValue(ctx, caughtSignalKey{}, &caught)
	var deferredHint func()
	ctx = context.WithValue(ctx, deferredHintKey{}, &deferredHint)
	// One skills hints value per run; see skills.WithHintsMemo.
	ctx = skills.WithHintsMemo(ctx)
	// slog.SetDefault also redirects the standard log package, so both are restored.
	prevLogger := slog.Default()
	prevLogWriter, prevLogFlags := stdlog.Writer(), stdlog.Flags()
	defer func() {
		logFile.close()
		slog.SetDefault(prevLogger)
		stdlog.SetOutput(prevLogWriter)
		stdlog.SetFlags(prevLogFlags)
		// Auto-install caught a signal after the command succeeded. Now that
		// the telemetry is uploaded and the log file is closed, the CLI ends by
		// it, so that the shell stops the script, loop or command list that ran
		// the command, as it would have without auto-install.
		if caught != nil {
			raise(caught)
		}
	}()

	// Run the command
	cmd, err = cmd.ExecuteContextC(ctx)
	if err != nil && !errors.Is(err, ErrAlreadyPrinted) {
		err = describeEmptyAPIError(err)
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
		// The message of an API error is the server's text.
		fmt.Fprintf(cmd.ErrOrStderr(), "Error: %s\n", textutil.EscapeControl(err.Error()))
	}

	// After auto-install, which runs only when the command succeeded, and
	// before the telemetry upload, which sends its hint-shown event.
	if deferredHint != nil {
		deferredHint()
	}

	exitCode := ExitStatus(err)

	// Log exit status and error
	// We only log if logger initialization succeeded and is stored in command
	// context
	if logger, ok := log.FromContext(cmd.Context()); ok {
		if err == nil {
			logger.Info("completed execution",
				slog.String("exit_code", "0"))
		} else if errors.Is(err, ErrAlreadyPrinted) {
			logger.Debug("failed execution",
				slog.String("exit_code", strconv.Itoa(exitCode)),
			)
		} else {
			logger.Info("failed execution",
				slog.String("exit_code", strconv.Itoa(exitCode)),
				slog.String("error", err.Error()),
			)
		}
	}

	ctx = cmd.Context()

	telemetryErr := telemetry.Upload(cmd.Context(), TelemetryExecutionContext(cmd, startTime, exitCode))
	if telemetryErr != nil {
		log.Infof(ctx, "telemetry upload failed: %s", telemetryErr)
	}

	return err
}

// TelemetryExecutionContext returns the execution context that a telemetry upload of cmd's run carries. start is when the
// run started, and exitCode is its exit status, 0 for a run that is still going. Execute uploads with it at exit, and a
// command that uploads before it exits uses it too.
func TelemetryExecutionContext(cmd *cobra.Command, start time.Time, exitCode int) protos.ExecutionContext {
	ctx := cmd.Context()
	return protos.ExecutionContext{
		CmdExecID:        cmdctx.ExecId(ctx),
		StateLineageHash: telemetry.StateLineageHash(ctx),
		Version:          build.GetInfo().Version,
		Command:          commandString(cmd),
		OperatingSystem:  runtime.GOOS,
		DbrVersion:       dbr.RuntimeVersion(ctx).String(),
		ExecutionTimeMs:  time.Since(start).Milliseconds(),
		ExitCode:         int64(exitCode),
	}
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

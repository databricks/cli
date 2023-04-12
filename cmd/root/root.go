package root

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/databricks/bricks/internal/build"
	"github.com/databricks/bricks/libs/log"
	"github.com/spf13/cobra"
	"golang.org/x/exp/slog"
)

// RootCmd represents the base command when called without any subcommands
var RootCmd = &cobra.Command{
	Use:   "bricks",
	Short: "Databricks project lifecycle management",
	Long:  `Where's "data"? Secured by the unity catalog. Projects build lifecycle is secured by bricks`,

	// Cobra prints the usage string to stderr if a command returns an error.
	// This usage string should only be displayed if an invalid combination of flags
	// is specified and not when runtime errors occur (e.g. resource not found).
	// The usage string is include in [flagErrorFunc] for flag errors only.
	SilenceUsage: true,

	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()

		// Configure default logger.
		ctx, err := initializeLogger(ctx)
		if err != nil {
			return err
		}

		// Update root command's context to include the initialized logger. This
		// is used to log exit codes and any error once execution of the command
		// is completed
		cmd.Root().SetContext(ctx)

		logger := log.GetLogger(ctx)
		logger.Info("start",
			slog.String("version", build.GetInfo().Version),
			slog.String("args", fmt.Sprintf("[%s]", strings.Join(os.Args, ", "))))

		// Configure progress logger
		ctx, err = initializeProgressLogger(ctx)
		if err != nil {
			return err
		}

		// Configure our user agent with the command that's about to be executed.
		ctx = withCommandInUserAgent(ctx, cmd)
		ctx = withUpstreamInUserAgent(ctx)
		cmd.SetContext(ctx)
		return nil
	},
	PersistentPostRun: func(cmd *cobra.Command, args []string) {
		log.Infof(cmd.Context(), "completed command execution")
	},
}

// Wrap flag errors to include the usage string.
func flagErrorFunc(c *cobra.Command, err error) error {
	return fmt.Errorf("%w\n\n%s", err, c.UsageString())
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	// TODO: deferred panic recovery
	ctx := context.Background()

	err := RootCmd.ExecuteContext(ctx)
	if err != nil {
		logger, ok := log.FromContext(RootCmd.Context())
		// We only log the error if logger initialization succeeded
		if ok {
			logger.Info("command execution failed",
				slog.String("exit_code", "1"),
				slog.String("error", err.Error()))
		}
		os.Exit(1)
	}
}

func init() {
	RootCmd.SetFlagErrorFunc(flagErrorFunc)

	// The VS Code extension passes `-v` in debug mode and must be changed
	// to use the new flags in `./logger.go` prior to removing this flag.
	RootCmd.PersistentFlags().BoolP("verbose", "v", false, "")
	RootCmd.PersistentFlags().MarkHidden("verbose")
}

package root

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"
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

	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		ctx := cmd.Context()

		// Configure default logger.
		ctx = initializeLogger(ctx, cmd)

		// Configure our user agent with the command that's about to be executed.
		ctx = withCommandInUserAgent(ctx, cmd)
		ctx = withUpstreamInUserAgent(ctx)
		cmd.SetContext(ctx)
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
		os.Exit(1)
	}
}

func init() {
	RootCmd.SetFlagErrorFunc(flagErrorFunc)
	// flags available for every child command
}

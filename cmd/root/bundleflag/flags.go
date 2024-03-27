package bundleflag

import (
	"github.com/databricks/cli/bundle/env"
	"github.com/spf13/cobra"
)

// Target returns the name of the target to operate in.
func Target(cmd *cobra.Command) (value string) {
	// The command line flag takes precedence.
	flag := cmd.Flag("target")
	if flag != nil {
		value = flag.Value.String()
		if value != "" {
			return
		}
	}

	oldFlag := cmd.Flag("environment")
	if oldFlag != nil {
		value = oldFlag.Value.String()
		if value != "" {
			return
		}
	}

	// If it's not set, use the environment variable.
	target, _ := env.Target(cmd.Context())
	return target
}

func Init(cmd *cobra.Command) {
	initTargetFlag(cmd)
	initEnvironmentFlag(cmd)
}

func initTargetFlag(cmd *cobra.Command) {
	// To operate in the context of a bundle, all commands must take an "target" parameter.
	cmd.PersistentFlags().StringP("target", "t", "", "bundle target to use (if applicable)")
	cmd.RegisterFlagCompletionFunc("target", targetCompletion)
}

// DEPRECATED flag
func initEnvironmentFlag(cmd *cobra.Command) {
	// To operate in the context of a bundle, all commands must take an "environment" parameter.
	cmd.PersistentFlags().StringP("environment", "e", "", "bundle target to use (if applicable)")
	cmd.PersistentFlags().MarkDeprecated("environment", "use --target flag instead")
	cmd.RegisterFlagCompletionFunc("environment", targetCompletion)
}

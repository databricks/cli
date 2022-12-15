package bundle

import (
	"os"

	"github.com/spf13/cobra"
)

const envName = "DATABRICKS_BUNDLE_ENV"

// getEnvironment returns the name of the environment to operate in.
func getEnvironment(cmd *cobra.Command) (value string) {
	// The command line flag takes precedence.
	flag := cmd.Flag("environment")
	if flag != nil {
		value = flag.Value.String()
		if value != "" {
			return
		}
	}

	// If it's not set, use the environment variable.
	return os.Getenv(envName)
}

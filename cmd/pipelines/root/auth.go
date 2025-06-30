package root

import (
	"github.com/databricks/cli/libs/databrickscfg/profile"
	"github.com/spf13/cobra"
)

func initProfileFlag(cmd *cobra.Command) {
	cmd.PersistentFlags().StringP("profile", "p", "", "~/.databrickscfg profile")
	cmd.RegisterFlagCompletionFunc("profile", profile.ProfileCompletion)
}

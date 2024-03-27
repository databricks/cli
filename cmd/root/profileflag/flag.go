package profileflag

import (
	"github.com/databricks/cli/libs/databrickscfg"
	"github.com/spf13/cobra"
)

func Init(cmd *cobra.Command) {
	cmd.PersistentFlags().StringP("profile", "p", "", "~/.databrickscfg profile")
	cmd.RegisterFlagCompletionFunc("profile", databrickscfg.ProfileCompletion)
}

func Value(cmd *cobra.Command) (string, bool) {
	profileFlag := cmd.Flag("profile")
	if profileFlag == nil {
		return "", false
	}
	value := profileFlag.Value.String()
	return value, value != ""
}

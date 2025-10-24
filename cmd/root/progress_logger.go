package root

import (
	"github.com/spf13/cobra"
)

func initProgressLoggerFlag(cmd *cobra.Command, logFlags *logFlags) {
	flags := cmd.PersistentFlags()
	_ = flags.String("progress-format", "", "format for progress logs")
	flags.MarkHidden("progress-format")
	flags.MarkDeprecated("progress-format", "this flag is no longer functional")
}

package genie

import (
	geniecmd "github.com/databricks/cli/cmd/genie"
	"github.com/spf13/cobra"
)

func init() {
	cmdOverrides = append(cmdOverrides, func(cmd *cobra.Command) {
		// "ask" is a hand-written streaming data-question command, not part of the
		// generated Genie spaces/conversations API surface; attach it to the same
		// top-level "genie" group so it runs as `databricks genie ask`.
		cmd.AddCommand(geniecmd.NewAskCmd())
	})
}

package selftest

import "github.com/spf13/cobra"

func newPanic() *cobra.Command {
	return &cobra.Command{
		Use: "panic",
		Run: func(cmd *cobra.Command, args []string) {
			panic("the databricks selftest panic command always panics")
		},
	}
}

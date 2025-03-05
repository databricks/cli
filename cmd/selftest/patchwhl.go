package selftest

import (
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/cli/libs/patchwheel"
	"github.com/spf13/cobra"
)

func newPatchWhl() *cobra.Command {
	return &cobra.Command{
		Use: "patchwhl",
		Run: func(cmd *cobra.Command, args []string) {
			ctx := cmd.Context()
			for _, arg := range args {
				out, err := patchwheel.PatchWheel(ctx, arg, ".")
				if err != nil {
					log.Warnf(ctx, "Failed to patch whl: %s: %s", arg, err)
				} else {
					log.Warnf(ctx, "Patched whl: %s -> %s", arg, out)
				}
			}
		},
	}
}

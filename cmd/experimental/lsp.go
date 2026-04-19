package experimental

import (
	"github.com/databricks/cli/bundle/lsp"
	"github.com/databricks/cli/cmd/root"
	"github.com/spf13/cobra"
)

func newBundleLspCommand() *cobra.Command {
	var target string

	cmd := &cobra.Command{
		Use:   "bundle-lsp",
		Short: "Start a Language Server Protocol server for bundle files",
		Args:  root.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			srv := lsp.NewServer()
			if target != "" {
				srv.SetTarget(target)
			}
			return srv.Run(cmd.Context())
		},
	}

	cmd.Flags().StringVar(&target, "target", "", "Bundle target to use for resource resolution")

	return cmd
}

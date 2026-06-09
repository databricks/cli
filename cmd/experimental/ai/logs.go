package ai

import (
	"github.com/databricks/cli/cmd/root"
	"github.com/spf13/cobra"
)

func newLogsCommand() *cobra.Command {
	var (
		node       int
		lines      int
		retry      int
		downloadTo string
		review     bool
	)

	cmd := &cobra.Command{
		Use:   "logs RUN_ID",
		Args:  root.ExactArgs(1),
		Short: "Stream or fetch logs for a run",
		Long:  `Stream logs from an active run, or fetch logs from a completed run.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return notImplemented("logs")
		},
	}

	cmd.Flags().IntVar(&node, "node", 0, "Fetch logs from this node")
	cmd.Flags().IntVar(&lines, "lines", 10000, "For completed runs, print the last N lines")
	cmd.Flags().IntVar(&retry, "retry", -1, "View logs from a specific retry attempt (default: latest)")
	cmd.Flags().StringVar(&downloadTo, "download-to", "", "Download all logs to this directory instead of printing")
	cmd.Flags().BoolVar(&review, "review", false, "Download logs from all nodes and filter for error signatures")

	return cmd
}

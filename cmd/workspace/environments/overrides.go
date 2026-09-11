package environments

import (
	environmentsCli "github.com/databricks/cli/cmd/environments"
	"github.com/spf13/cobra"
)

func init() {
	cmdOverrides = append(cmdOverrides, func(cmd *cobra.Command) {
		// Attach the hand-written local-provisioning commands (e.g. setup-local)
		// to the auto-generated "environments" group. The group intentionally
		// spans server-side environment-resource APIs and local provisioning, so
		// a local-install verb belongs here (spec §naming). This mirrors how
		// cmd/apps extends the generated apps group.
		for _, c := range environmentsCli.Commands() {
			cmd.AddCommand(c)
		}
	})
}

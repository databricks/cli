package pipelines

import (
	bundlecmd "github.com/databricks/cli/cmd/bundle"
	"github.com/spf13/cobra"
)

func Deploy() *cobra.Command {
	return bundlecmd.NewDeployCommand()
}

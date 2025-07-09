package pipelines

import (
	"github.com/databricks/cli/cmd/auth"
	"github.com/spf13/cobra"
)

func authCommand() *cobra.Command {
	return auth.New()
}

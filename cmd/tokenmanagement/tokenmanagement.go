package tokenmanagement

import (
	token_management "github.com/databricks/bricks/cmd/tokenmanagement/token-management"
	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use: "tokenmanagement",
}

func init() {

	Cmd.AddCommand(token_management.Cmd)
}

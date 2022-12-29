package auth

import (
	"github.com/databricks/bricks/cmd/root"
	"github.com/spf13/cobra"
)

var authCmd = &cobra.Command{
	Use:   "auth",
	Short: "Authentication related commands",
}

func init() {
	root.RootCmd.AddCommand(authCmd)
}

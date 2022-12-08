package ipaccesslists

import (
	ip_access_lists "github.com/databricks/bricks/cmd/ipaccesslists/ip-access-lists"
	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use: "ipaccesslists",
}

func init() {
	Cmd.PersistentFlags().String("profile", "", "~/.databrickscfg profile")

	Cmd.AddCommand(ip_access_lists.Cmd)
}

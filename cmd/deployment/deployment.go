package deployment

import (
	"github.com/databricks/bricks/cmd/deployment/credentials"
	encryption_keys "github.com/databricks/bricks/cmd/deployment/encryption-keys"
	"github.com/databricks/bricks/cmd/deployment/networks"
	private_access "github.com/databricks/bricks/cmd/deployment/private-access"
	"github.com/databricks/bricks/cmd/deployment/storage"
	vpc_endpoints "github.com/databricks/bricks/cmd/deployment/vpc-endpoints"
	"github.com/databricks/bricks/cmd/deployment/workspaces"
	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use: "deployment",
}

func init() {
	Cmd.PersistentFlags().String("profile", "", "~/.databrickscfg profile")

	Cmd.AddCommand(credentials.Cmd)
	Cmd.AddCommand(encryption_keys.Cmd)
	Cmd.AddCommand(networks.Cmd)
	Cmd.AddCommand(private_access.Cmd)
	Cmd.AddCommand(storage.Cmd)
	Cmd.AddCommand(vpc_endpoints.Cmd)
	Cmd.AddCommand(workspaces.Cmd)
}

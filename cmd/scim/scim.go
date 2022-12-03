package scim

import (
	account_groups "github.com/databricks/bricks/cmd/scim/account-groups"
	account_service_principals "github.com/databricks/bricks/cmd/scim/account-service-principals"
	account_users "github.com/databricks/bricks/cmd/scim/account-users"
	current_user "github.com/databricks/bricks/cmd/scim/current-user"
	"github.com/databricks/bricks/cmd/scim/groups"
	service_principals "github.com/databricks/bricks/cmd/scim/service-principals"
	"github.com/databricks/bricks/cmd/scim/users"
	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use: "scim",
}

func init() {

	Cmd.AddCommand(account_groups.Cmd)
	Cmd.AddCommand(account_service_principals.Cmd)
	Cmd.AddCommand(account_users.Cmd)
	Cmd.AddCommand(current_user.Cmd)
	Cmd.AddCommand(groups.Cmd)
	Cmd.AddCommand(service_principals.Cmd)
	Cmd.AddCommand(users.Cmd)
}

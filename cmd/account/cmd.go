// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package account

import (
	"github.com/databricks/bricks/cmd/root"
	"github.com/spf13/cobra"

	billable_usage "github.com/databricks/bricks/cmd/account/billable-usage"
	budgets "github.com/databricks/bricks/cmd/account/budgets"
	credentials "github.com/databricks/bricks/cmd/account/credentials"
	custom_app_integration "github.com/databricks/bricks/cmd/account/custom-app-integration"
	encryption_keys "github.com/databricks/bricks/cmd/account/encryption-keys"
	account_groups "github.com/databricks/bricks/cmd/account/groups"
	account_ip_access_lists "github.com/databricks/bricks/cmd/account/ip-access-lists"
	log_delivery "github.com/databricks/bricks/cmd/account/log-delivery"
	account_metastore_assignments "github.com/databricks/bricks/cmd/account/metastore-assignments"
	account_metastores "github.com/databricks/bricks/cmd/account/metastores"
	networks "github.com/databricks/bricks/cmd/account/networks"
	o_auth_enrollment "github.com/databricks/bricks/cmd/account/o-auth-enrollment"
	private_access "github.com/databricks/bricks/cmd/account/private-access"
	published_app_integration "github.com/databricks/bricks/cmd/account/published-app-integration"
	account_service_principals "github.com/databricks/bricks/cmd/account/service-principals"
	storage "github.com/databricks/bricks/cmd/account/storage"
	account_storage_credentials "github.com/databricks/bricks/cmd/account/storage-credentials"
	account_users "github.com/databricks/bricks/cmd/account/users"
	vpc_endpoints "github.com/databricks/bricks/cmd/account/vpc-endpoints"
	workspace_assignment "github.com/databricks/bricks/cmd/account/workspace-assignment"
	workspaces "github.com/databricks/bricks/cmd/account/workspaces"
)

var accountCmd = &cobra.Command{
	Use:   "account",
	Short: `Databricks Account Commands`,
}

func init() {
	root.RootCmd.AddCommand(accountCmd)

	accountCmd.AddCommand(billable_usage.Cmd)
	accountCmd.AddCommand(budgets.Cmd)
	accountCmd.AddCommand(credentials.Cmd)
	accountCmd.AddCommand(custom_app_integration.Cmd)
	accountCmd.AddCommand(encryption_keys.Cmd)
	accountCmd.AddCommand(account_groups.Cmd)
	accountCmd.AddCommand(account_ip_access_lists.Cmd)
	accountCmd.AddCommand(log_delivery.Cmd)
	accountCmd.AddCommand(account_metastore_assignments.Cmd)
	accountCmd.AddCommand(account_metastores.Cmd)
	accountCmd.AddCommand(networks.Cmd)
	accountCmd.AddCommand(o_auth_enrollment.Cmd)
	accountCmd.AddCommand(private_access.Cmd)
	accountCmd.AddCommand(published_app_integration.Cmd)
	accountCmd.AddCommand(account_service_principals.Cmd)
	accountCmd.AddCommand(storage.Cmd)
	accountCmd.AddCommand(account_storage_credentials.Cmd)
	accountCmd.AddCommand(account_users.Cmd)
	accountCmd.AddCommand(vpc_endpoints.Cmd)
	accountCmd.AddCommand(workspace_assignment.Cmd)
	accountCmd.AddCommand(workspaces.Cmd)
}

// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package account

import (
	"github.com/databricks/cli/cmd/root"
	"github.com/spf13/cobra"

	account_access_control "github.com/databricks/cli/cmd/account/access-control"
	billable_usage "github.com/databricks/cli/cmd/account/billable-usage"
	budgets "github.com/databricks/cli/cmd/account/budgets"
	credentials "github.com/databricks/cli/cmd/account/credentials"
	custom_app_integration "github.com/databricks/cli/cmd/account/custom-app-integration"
	encryption_keys "github.com/databricks/cli/cmd/account/encryption-keys"
	account_groups "github.com/databricks/cli/cmd/account/groups"
	account_ip_access_lists "github.com/databricks/cli/cmd/account/ip-access-lists"
	log_delivery "github.com/databricks/cli/cmd/account/log-delivery"
	account_metastore_assignments "github.com/databricks/cli/cmd/account/metastore-assignments"
	account_metastores "github.com/databricks/cli/cmd/account/metastores"
	networks "github.com/databricks/cli/cmd/account/networks"
	o_auth_enrollment "github.com/databricks/cli/cmd/account/o-auth-enrollment"
	private_access "github.com/databricks/cli/cmd/account/private-access"
	published_app_integration "github.com/databricks/cli/cmd/account/published-app-integration"
	service_principal_secrets "github.com/databricks/cli/cmd/account/service-principal-secrets"
	account_service_principals "github.com/databricks/cli/cmd/account/service-principals"
	account_settings "github.com/databricks/cli/cmd/account/settings"
	storage "github.com/databricks/cli/cmd/account/storage"
	account_storage_credentials "github.com/databricks/cli/cmd/account/storage-credentials"
	account_users "github.com/databricks/cli/cmd/account/users"
	vpc_endpoints "github.com/databricks/cli/cmd/account/vpc-endpoints"
	workspace_assignment "github.com/databricks/cli/cmd/account/workspace-assignment"
	workspaces "github.com/databricks/cli/cmd/account/workspaces"
)

var accountCmd = &cobra.Command{
	Use:   "account",
	Short: `Databricks Account Commands`,
}

func init() {
	root.RootCmd.AddCommand(accountCmd)

	accountCmd.AddCommand(account_access_control.Cmd)
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
	accountCmd.AddCommand(service_principal_secrets.Cmd)
	accountCmd.AddCommand(account_service_principals.Cmd)
	accountCmd.AddCommand(account_settings.Cmd)
	accountCmd.AddCommand(storage.Cmd)
	accountCmd.AddCommand(account_storage_credentials.Cmd)
	accountCmd.AddCommand(account_users.Cmd)
	accountCmd.AddCommand(vpc_endpoints.Cmd)
	accountCmd.AddCommand(workspace_assignment.Cmd)
	accountCmd.AddCommand(workspaces.Cmd)
}

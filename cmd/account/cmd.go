// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package account

import (
	"github.com/spf13/cobra"

	account_access_control "github.com/databricks/cli/cmd/account/access-control"
	billable_usage "github.com/databricks/cli/cmd/account/billable-usage"
	budget_policy "github.com/databricks/cli/cmd/account/budget-policy"
	budgets "github.com/databricks/cli/cmd/account/budgets"
	credentials "github.com/databricks/cli/cmd/account/credentials"
	custom_app_integration "github.com/databricks/cli/cmd/account/custom-app-integration"
	encryption_keys "github.com/databricks/cli/cmd/account/encryption-keys"
	account_federation_policy "github.com/databricks/cli/cmd/account/federation-policy"
	account_groups "github.com/databricks/cli/cmd/account/groups"
	account_ip_access_lists "github.com/databricks/cli/cmd/account/ip-access-lists"
	log_delivery "github.com/databricks/cli/cmd/account/log-delivery"
	account_metastore_assignments "github.com/databricks/cli/cmd/account/metastore-assignments"
	account_metastores "github.com/databricks/cli/cmd/account/metastores"
	network_connectivity "github.com/databricks/cli/cmd/account/network-connectivity"
	network_policies "github.com/databricks/cli/cmd/account/network-policies"
	networks "github.com/databricks/cli/cmd/account/networks"
	o_auth_published_apps "github.com/databricks/cli/cmd/account/o-auth-published-apps"
	private_access "github.com/databricks/cli/cmd/account/private-access"
	published_app_integration "github.com/databricks/cli/cmd/account/published-app-integration"
	service_principal_federation_policy "github.com/databricks/cli/cmd/account/service-principal-federation-policy"
	service_principal_secrets "github.com/databricks/cli/cmd/account/service-principal-secrets"
	account_service_principals "github.com/databricks/cli/cmd/account/service-principals"
	account_settings "github.com/databricks/cli/cmd/account/settings"
	storage "github.com/databricks/cli/cmd/account/storage"
	account_storage_credentials "github.com/databricks/cli/cmd/account/storage-credentials"
	usage_dashboards "github.com/databricks/cli/cmd/account/usage-dashboards"
	account_users "github.com/databricks/cli/cmd/account/users"
	vpc_endpoints "github.com/databricks/cli/cmd/account/vpc-endpoints"
	workspace_assignment "github.com/databricks/cli/cmd/account/workspace-assignment"
	workspace_network_configuration "github.com/databricks/cli/cmd/account/workspace-network-configuration"
	workspaces "github.com/databricks/cli/cmd/account/workspaces"
)

func New() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "account",
		Short: `Databricks Account Commands`,
	}

	cmd.AddCommand(account_access_control.New())
	cmd.AddCommand(billable_usage.New())
	cmd.AddCommand(budget_policy.New())
	cmd.AddCommand(credentials.New())
	cmd.AddCommand(custom_app_integration.New())
	cmd.AddCommand(encryption_keys.New())
	cmd.AddCommand(account_federation_policy.New())
	cmd.AddCommand(account_groups.New())
	cmd.AddCommand(account_ip_access_lists.New())
	cmd.AddCommand(log_delivery.New())
	cmd.AddCommand(account_metastore_assignments.New())
	cmd.AddCommand(account_metastores.New())
	cmd.AddCommand(network_connectivity.New())
	cmd.AddCommand(network_policies.New())
	cmd.AddCommand(networks.New())
	cmd.AddCommand(o_auth_published_apps.New())
	cmd.AddCommand(private_access.New())
	cmd.AddCommand(published_app_integration.New())
	cmd.AddCommand(service_principal_federation_policy.New())
	cmd.AddCommand(service_principal_secrets.New())
	cmd.AddCommand(account_service_principals.New())
	cmd.AddCommand(account_settings.New())
	cmd.AddCommand(storage.New())
	cmd.AddCommand(account_storage_credentials.New())
	cmd.AddCommand(usage_dashboards.New())
	cmd.AddCommand(account_users.New())
	cmd.AddCommand(vpc_endpoints.New())
	cmd.AddCommand(workspace_assignment.New())
	cmd.AddCommand(workspace_network_configuration.New())
	cmd.AddCommand(workspaces.New())
	cmd.AddCommand(budgets.New())

	// Register all groups with the parent command.
	groups := Groups()
	for i := range groups {
		cmd.AddGroup(&groups[i])
	}

	return cmd
}

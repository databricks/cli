// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package account

import (
	"context"

	"github.com/databricks/cli/cmd/root"
	"github.com/spf13/cobra"

	account_access_control "github.com/databricks/cli/cmd/account/access-control"
	billable_usage "github.com/databricks/cli/cmd/account/billable-usage"
	budget_policy "github.com/databricks/cli/cmd/account/budget-policy"
	budgets "github.com/databricks/cli/cmd/account/budgets"
	credentials "github.com/databricks/cli/cmd/account/credentials"
	custom_app_integration "github.com/databricks/cli/cmd/account/custom-app-integration"
	disaster_recovery "github.com/databricks/cli/cmd/account/disaster-recovery"
	encryption_keys "github.com/databricks/cli/cmd/account/encryption-keys"
	endpoints "github.com/databricks/cli/cmd/account/endpoints"
	account_federation_policy "github.com/databricks/cli/cmd/account/federation-policy"
	account_groups_v2 "github.com/databricks/cli/cmd/account/groups-v2"
	account_iam_v2 "github.com/databricks/cli/cmd/account/iam-v2"
	account_ip_access_lists "github.com/databricks/cli/cmd/account/ip-access-lists"
	log_delivery "github.com/databricks/cli/cmd/account/log-delivery"
	account_metastore_assignments "github.com/databricks/cli/cmd/account/metastore-assignments"
	account_metastores "github.com/databricks/cli/cmd/account/metastores"
	network_connectivity "github.com/databricks/cli/cmd/account/network-connectivity"
	network_policies "github.com/databricks/cli/cmd/account/network-policies"
	networks "github.com/databricks/cli/cmd/account/networks"
	o_auth_published_apps "github.com/databricks/cli/cmd/account/o-auth-published-apps"
	private_access "github.com/databricks/cli/cmd/account/private-access"
	private_network_gateways "github.com/databricks/cli/cmd/account/private-network-gateways"
	published_app_integration "github.com/databricks/cli/cmd/account/published-app-integration"
	service_principal_federation_policy "github.com/databricks/cli/cmd/account/service-principal-federation-policy"
	service_principal_secrets "github.com/databricks/cli/cmd/account/service-principal-secrets"
	account_service_principals_v2 "github.com/databricks/cli/cmd/account/service-principals-v2"
	account_settings "github.com/databricks/cli/cmd/account/settings"
	account_settings_v2 "github.com/databricks/cli/cmd/account/settings-v2"
	storage "github.com/databricks/cli/cmd/account/storage"
	account_storage_credentials "github.com/databricks/cli/cmd/account/storage-credentials"
	usage_dashboards "github.com/databricks/cli/cmd/account/usage-dashboards"
	account_users_v2 "github.com/databricks/cli/cmd/account/users-v2"
	vpc_endpoints "github.com/databricks/cli/cmd/account/vpc-endpoints"
	workspace_assignment "github.com/databricks/cli/cmd/account/workspace-assignment"
	workspace_network_configuration "github.com/databricks/cli/cmd/account/workspace-network-configuration"
	workspaces "github.com/databricks/cli/cmd/account/workspaces"

	account_groups "github.com/databricks/cli/cmd/account/groups"
	account_service_principals "github.com/databricks/cli/cmd/account/service-principals"
	account_users "github.com/databricks/cli/cmd/account/users"
)

// Spec declares the account command.
var Spec = root.CommandSpec{
	Use:   "account",
	Short: `Databricks Account Commands`,
	Build: buildCommand,
}

func buildCommand(ctx context.Context, cmd *cobra.Command) {
	cmd.AddCommand(root.NewCommand(ctx, account_access_control.Spec))
	cmd.AddCommand(root.NewCommand(ctx, billable_usage.Spec))
	cmd.AddCommand(root.NewCommand(ctx, budget_policy.Spec))
	cmd.AddCommand(root.NewCommand(ctx, budgets.Spec))
	cmd.AddCommand(root.NewCommand(ctx, credentials.Spec))
	cmd.AddCommand(root.NewCommand(ctx, custom_app_integration.Spec))
	cmd.AddCommand(root.NewCommand(ctx, disaster_recovery.Spec))
	cmd.AddCommand(root.NewCommand(ctx, encryption_keys.Spec))
	cmd.AddCommand(root.NewCommand(ctx, endpoints.Spec))
	cmd.AddCommand(root.NewCommand(ctx, account_federation_policy.Spec))
	cmd.AddCommand(root.NewCommand(ctx, account_groups_v2.Spec))
	cmd.AddCommand(root.NewCommand(ctx, account_iam_v2.Spec))
	cmd.AddCommand(root.NewCommand(ctx, account_ip_access_lists.Spec))
	cmd.AddCommand(root.NewCommand(ctx, log_delivery.Spec))
	cmd.AddCommand(root.NewCommand(ctx, account_metastore_assignments.Spec))
	cmd.AddCommand(root.NewCommand(ctx, account_metastores.Spec))
	cmd.AddCommand(root.NewCommand(ctx, network_connectivity.Spec))
	cmd.AddCommand(root.NewCommand(ctx, network_policies.Spec))
	cmd.AddCommand(root.NewCommand(ctx, networks.Spec))
	cmd.AddCommand(root.NewCommand(ctx, o_auth_published_apps.Spec))
	cmd.AddCommand(root.NewCommand(ctx, private_access.Spec))
	cmd.AddCommand(root.NewCommand(ctx, private_network_gateways.Spec))
	cmd.AddCommand(root.NewCommand(ctx, published_app_integration.Spec))
	cmd.AddCommand(root.NewCommand(ctx, service_principal_federation_policy.Spec))
	cmd.AddCommand(root.NewCommand(ctx, service_principal_secrets.Spec))
	cmd.AddCommand(root.NewCommand(ctx, account_service_principals_v2.Spec))
	cmd.AddCommand(root.NewCommand(ctx, account_settings.Spec))
	cmd.AddCommand(root.NewCommand(ctx, account_settings_v2.Spec))
	cmd.AddCommand(root.NewCommand(ctx, storage.Spec))
	cmd.AddCommand(root.NewCommand(ctx, account_storage_credentials.Spec))
	cmd.AddCommand(root.NewCommand(ctx, usage_dashboards.Spec))
	cmd.AddCommand(root.NewCommand(ctx, account_users_v2.Spec))
	cmd.AddCommand(root.NewCommand(ctx, vpc_endpoints.Spec))
	cmd.AddCommand(root.NewCommand(ctx, workspace_assignment.Spec))
	cmd.AddCommand(root.NewCommand(ctx, workspace_network_configuration.Spec))
	cmd.AddCommand(root.NewCommand(ctx, workspaces.Spec))

	cmd.AddCommand(root.NewCommand(ctx, account_groups.Spec))
	cmd.AddCommand(root.NewCommand(ctx, account_service_principals.Spec))
	cmd.AddCommand(root.NewCommand(ctx, account_users.Spec))
}

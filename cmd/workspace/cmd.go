// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package workspace

import (
	alerts "github.com/databricks/cli/cmd/workspace/alerts"
	alerts_legacy "github.com/databricks/cli/cmd/workspace/alerts-legacy"
	apps "github.com/databricks/cli/cmd/workspace/apps"
	artifact_allowlists "github.com/databricks/cli/cmd/workspace/artifact-allowlists"
	catalogs "github.com/databricks/cli/cmd/workspace/catalogs"
	clean_rooms "github.com/databricks/cli/cmd/workspace/clean-rooms"
	cluster_policies "github.com/databricks/cli/cmd/workspace/cluster-policies"
	clusters "github.com/databricks/cli/cmd/workspace/clusters"
	connections "github.com/databricks/cli/cmd/workspace/connections"
	consumer_fulfillments "github.com/databricks/cli/cmd/workspace/consumer-fulfillments"
	consumer_installations "github.com/databricks/cli/cmd/workspace/consumer-installations"
	consumer_listings "github.com/databricks/cli/cmd/workspace/consumer-listings"
	consumer_personalization_requests "github.com/databricks/cli/cmd/workspace/consumer-personalization-requests"
	consumer_providers "github.com/databricks/cli/cmd/workspace/consumer-providers"
	credentials_manager "github.com/databricks/cli/cmd/workspace/credentials-manager"
	current_user "github.com/databricks/cli/cmd/workspace/current-user"
	dashboard_widgets "github.com/databricks/cli/cmd/workspace/dashboard-widgets"
	dashboards "github.com/databricks/cli/cmd/workspace/dashboards"
	data_sources "github.com/databricks/cli/cmd/workspace/data-sources"
	experiments "github.com/databricks/cli/cmd/workspace/experiments"
	external_locations "github.com/databricks/cli/cmd/workspace/external-locations"
	functions "github.com/databricks/cli/cmd/workspace/functions"
	genie "github.com/databricks/cli/cmd/workspace/genie"
	git_credentials "github.com/databricks/cli/cmd/workspace/git-credentials"
	global_init_scripts "github.com/databricks/cli/cmd/workspace/global-init-scripts"
	grants "github.com/databricks/cli/cmd/workspace/grants"
	groups "github.com/databricks/cli/cmd/workspace/groups"
	instance_pools "github.com/databricks/cli/cmd/workspace/instance-pools"
	instance_profiles "github.com/databricks/cli/cmd/workspace/instance-profiles"
	ip_access_lists "github.com/databricks/cli/cmd/workspace/ip-access-lists"
	jobs "github.com/databricks/cli/cmd/workspace/jobs"
	lakeview "github.com/databricks/cli/cmd/workspace/lakeview"
	libraries "github.com/databricks/cli/cmd/workspace/libraries"
	metastores "github.com/databricks/cli/cmd/workspace/metastores"
	model_registry "github.com/databricks/cli/cmd/workspace/model-registry"
	model_versions "github.com/databricks/cli/cmd/workspace/model-versions"
	notification_destinations "github.com/databricks/cli/cmd/workspace/notification-destinations"
	online_tables "github.com/databricks/cli/cmd/workspace/online-tables"
	permission_migration "github.com/databricks/cli/cmd/workspace/permission-migration"
	permissions "github.com/databricks/cli/cmd/workspace/permissions"
	pipelines "github.com/databricks/cli/cmd/workspace/pipelines"
	policy_compliance_for_clusters "github.com/databricks/cli/cmd/workspace/policy-compliance-for-clusters"
	policy_compliance_for_jobs "github.com/databricks/cli/cmd/workspace/policy-compliance-for-jobs"
	policy_families "github.com/databricks/cli/cmd/workspace/policy-families"
	provider_exchange_filters "github.com/databricks/cli/cmd/workspace/provider-exchange-filters"
	provider_exchanges "github.com/databricks/cli/cmd/workspace/provider-exchanges"
	provider_files "github.com/databricks/cli/cmd/workspace/provider-files"
	provider_listings "github.com/databricks/cli/cmd/workspace/provider-listings"
	provider_personalization_requests "github.com/databricks/cli/cmd/workspace/provider-personalization-requests"
	provider_provider_analytics_dashboards "github.com/databricks/cli/cmd/workspace/provider-provider-analytics-dashboards"
	provider_providers "github.com/databricks/cli/cmd/workspace/provider-providers"
	providers "github.com/databricks/cli/cmd/workspace/providers"
	quality_monitors "github.com/databricks/cli/cmd/workspace/quality-monitors"
	queries "github.com/databricks/cli/cmd/workspace/queries"
	queries_legacy "github.com/databricks/cli/cmd/workspace/queries-legacy"
	query_history "github.com/databricks/cli/cmd/workspace/query-history"
	query_visualizations "github.com/databricks/cli/cmd/workspace/query-visualizations"
	query_visualizations_legacy "github.com/databricks/cli/cmd/workspace/query-visualizations-legacy"
	recipient_activation "github.com/databricks/cli/cmd/workspace/recipient-activation"
	recipients "github.com/databricks/cli/cmd/workspace/recipients"
	registered_models "github.com/databricks/cli/cmd/workspace/registered-models"
	repos "github.com/databricks/cli/cmd/workspace/repos"
	resource_quotas "github.com/databricks/cli/cmd/workspace/resource-quotas"
	schemas "github.com/databricks/cli/cmd/workspace/schemas"
	secrets "github.com/databricks/cli/cmd/workspace/secrets"
	service_principals "github.com/databricks/cli/cmd/workspace/service-principals"
	serving_endpoints "github.com/databricks/cli/cmd/workspace/serving-endpoints"
	settings "github.com/databricks/cli/cmd/workspace/settings"
	shares "github.com/databricks/cli/cmd/workspace/shares"
	storage_credentials "github.com/databricks/cli/cmd/workspace/storage-credentials"
	system_schemas "github.com/databricks/cli/cmd/workspace/system-schemas"
	table_constraints "github.com/databricks/cli/cmd/workspace/table-constraints"
	tables "github.com/databricks/cli/cmd/workspace/tables"
	temporary_table_credentials "github.com/databricks/cli/cmd/workspace/temporary-table-credentials"
	token_management "github.com/databricks/cli/cmd/workspace/token-management"
	tokens "github.com/databricks/cli/cmd/workspace/tokens"
	users "github.com/databricks/cli/cmd/workspace/users"
	vector_search_endpoints "github.com/databricks/cli/cmd/workspace/vector-search-endpoints"
	vector_search_indexes "github.com/databricks/cli/cmd/workspace/vector-search-indexes"
	volumes "github.com/databricks/cli/cmd/workspace/volumes"
	warehouses "github.com/databricks/cli/cmd/workspace/warehouses"
	workspace "github.com/databricks/cli/cmd/workspace/workspace"
	workspace_bindings "github.com/databricks/cli/cmd/workspace/workspace-bindings"
	workspace_conf "github.com/databricks/cli/cmd/workspace/workspace-conf"
	"github.com/spf13/cobra"
)

func All() []*cobra.Command {
	var out []*cobra.Command

	out = append(out, alerts.New())
	out = append(out, alerts_legacy.New())
	out = append(out, apps.New())
	out = append(out, artifact_allowlists.New())
	out = append(out, catalogs.New())
	out = append(out, clean_rooms.New())
	out = append(out, cluster_policies.New())
	out = append(out, clusters.New())
	out = append(out, connections.New())
	out = append(out, consumer_fulfillments.New())
	out = append(out, consumer_installations.New())
	out = append(out, consumer_listings.New())
	out = append(out, consumer_personalization_requests.New())
	out = append(out, consumer_providers.New())
	out = append(out, credentials_manager.New())
	out = append(out, current_user.New())
	out = append(out, dashboard_widgets.New())
	out = append(out, dashboards.New())
	out = append(out, data_sources.New())
	out = append(out, experiments.New())
	out = append(out, external_locations.New())
	out = append(out, functions.New())
	out = append(out, genie.New())
	out = append(out, git_credentials.New())
	out = append(out, global_init_scripts.New())
	out = append(out, grants.New())
	out = append(out, groups.New())
	out = append(out, instance_pools.New())
	out = append(out, instance_profiles.New())
	out = append(out, ip_access_lists.New())
	out = append(out, jobs.New())
	out = append(out, lakeview.New())
	out = append(out, libraries.New())
	out = append(out, metastores.New())
	out = append(out, model_registry.New())
	out = append(out, model_versions.New())
	out = append(out, notification_destinations.New())
	out = append(out, online_tables.New())
	out = append(out, permission_migration.New())
	out = append(out, permissions.New())
	out = append(out, pipelines.New())
	out = append(out, policy_compliance_for_clusters.New())
	out = append(out, policy_compliance_for_jobs.New())
	out = append(out, policy_families.New())
	out = append(out, provider_exchange_filters.New())
	out = append(out, provider_exchanges.New())
	out = append(out, provider_files.New())
	out = append(out, provider_listings.New())
	out = append(out, provider_personalization_requests.New())
	out = append(out, provider_provider_analytics_dashboards.New())
	out = append(out, provider_providers.New())
	out = append(out, providers.New())
	out = append(out, quality_monitors.New())
	out = append(out, queries.New())
	out = append(out, queries_legacy.New())
	out = append(out, query_history.New())
	out = append(out, query_visualizations.New())
	out = append(out, query_visualizations_legacy.New())
	out = append(out, recipient_activation.New())
	out = append(out, recipients.New())
	out = append(out, registered_models.New())
	out = append(out, repos.New())
	out = append(out, resource_quotas.New())
	out = append(out, schemas.New())
	out = append(out, secrets.New())
	out = append(out, service_principals.New())
	out = append(out, serving_endpoints.New())
	out = append(out, settings.New())
	out = append(out, shares.New())
	out = append(out, storage_credentials.New())
	out = append(out, system_schemas.New())
	out = append(out, table_constraints.New())
	out = append(out, tables.New())
	out = append(out, temporary_table_credentials.New())
	out = append(out, token_management.New())
	out = append(out, tokens.New())
	out = append(out, users.New())
	out = append(out, vector_search_endpoints.New())
	out = append(out, vector_search_indexes.New())
	out = append(out, volumes.New())
	out = append(out, warehouses.New())
	out = append(out, workspace.New())
	out = append(out, workspace_bindings.New())
	out = append(out, workspace_conf.New())

	return out
}

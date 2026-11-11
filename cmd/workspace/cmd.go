// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package workspace

import (
	"github.com/databricks/cli/cmd/root"
	access_control "github.com/databricks/cli/cmd/workspace/access-control"
	agent_bricks "github.com/databricks/cli/cmd/workspace/agent-bricks"
	agent_kit "github.com/databricks/cli/cmd/workspace/agent-kit"
	ai_functions "github.com/databricks/cli/cmd/workspace/ai-functions"
	ai_gateway "github.com/databricks/cli/cmd/workspace/ai-gateway"
	ai_search "github.com/databricks/cli/cmd/workspace/ai-search"
	alerts "github.com/databricks/cli/cmd/workspace/alerts"
	alerts_legacy "github.com/databricks/cli/cmd/workspace/alerts-legacy"
	alerts_v2 "github.com/databricks/cli/cmd/workspace/alerts-v2"
	apps "github.com/databricks/cli/cmd/workspace/apps"
	apps_settings "github.com/databricks/cli/cmd/workspace/apps-settings"
	artifact_allowlists "github.com/databricks/cli/cmd/workspace/artifact-allowlists"
	bundle_deployments "github.com/databricks/cli/cmd/workspace/bundle-deployments"
	catalogs "github.com/databricks/cli/cmd/workspace/catalogs"
	clean_room_asset_revisions "github.com/databricks/cli/cmd/workspace/clean-room-asset-revisions"
	clean_room_assets "github.com/databricks/cli/cmd/workspace/clean-room-assets"
	clean_room_auto_approval_rules "github.com/databricks/cli/cmd/workspace/clean-room-auto-approval-rules"
	clean_room_task_runs "github.com/databricks/cli/cmd/workspace/clean-room-task-runs"
	clean_rooms "github.com/databricks/cli/cmd/workspace/clean-rooms"
	cluster_policies "github.com/databricks/cli/cmd/workspace/cluster-policies"
	clusters "github.com/databricks/cli/cmd/workspace/clusters"
	compute_migrations "github.com/databricks/cli/cmd/workspace/compute-migrations"
	connections "github.com/databricks/cli/cmd/workspace/connections"
	consumer_fulfillments "github.com/databricks/cli/cmd/workspace/consumer-fulfillments"
	consumer_installations "github.com/databricks/cli/cmd/workspace/consumer-installations"
	consumer_listings "github.com/databricks/cli/cmd/workspace/consumer-listings"
	consumer_personalization_requests "github.com/databricks/cli/cmd/workspace/consumer-personalization-requests"
	consumer_providers "github.com/databricks/cli/cmd/workspace/consumer-providers"
	credentials "github.com/databricks/cli/cmd/workspace/credentials"
	credentials_manager "github.com/databricks/cli/cmd/workspace/credentials-manager"
	current_user "github.com/databricks/cli/cmd/workspace/current-user"
	dashboard_widgets "github.com/databricks/cli/cmd/workspace/dashboard-widgets"
	dashboards "github.com/databricks/cli/cmd/workspace/dashboards"
	data_classification "github.com/databricks/cli/cmd/workspace/data-classification"
	data_quality "github.com/databricks/cli/cmd/workspace/data-quality"
	data_sources "github.com/databricks/cli/cmd/workspace/data-sources"
	database "github.com/databricks/cli/cmd/workspace/database"
	domains "github.com/databricks/cli/cmd/workspace/domains"
	entity_tag_assignments "github.com/databricks/cli/cmd/workspace/entity-tag-assignments"
	environments "github.com/databricks/cli/cmd/workspace/environments"
	experiments "github.com/databricks/cli/cmd/workspace/experiments"
	external_lineage "github.com/databricks/cli/cmd/workspace/external-lineage"
	external_locations "github.com/databricks/cli/cmd/workspace/external-locations"
	external_metadata "github.com/databricks/cli/cmd/workspace/external-metadata"
	feature_engineering "github.com/databricks/cli/cmd/workspace/feature-engineering"
	feature_store "github.com/databricks/cli/cmd/workspace/feature-store"
	forecasting "github.com/databricks/cli/cmd/workspace/forecasting"
	functions "github.com/databricks/cli/cmd/workspace/functions"
	genie "github.com/databricks/cli/cmd/workspace/genie"
	genie_agents "github.com/databricks/cli/cmd/workspace/genie-agents"
	git_credentials "github.com/databricks/cli/cmd/workspace/git-credentials"
	global_init_scripts "github.com/databricks/cli/cmd/workspace/global-init-scripts"
	grants "github.com/databricks/cli/cmd/workspace/grants"
	groups_v2 "github.com/databricks/cli/cmd/workspace/groups-v2"
	instance_pools "github.com/databricks/cli/cmd/workspace/instance-pools"
	instance_profiles "github.com/databricks/cli/cmd/workspace/instance-profiles"
	ip_access_lists "github.com/databricks/cli/cmd/workspace/ip-access-lists"
	jobs "github.com/databricks/cli/cmd/workspace/jobs"
	knowledge_assistants "github.com/databricks/cli/cmd/workspace/knowledge-assistants"
	lakeview "github.com/databricks/cli/cmd/workspace/lakeview"
	lakeview_embedded "github.com/databricks/cli/cmd/workspace/lakeview-embedded"
	lakewatch "github.com/databricks/cli/cmd/workspace/lakewatch"
	libraries "github.com/databricks/cli/cmd/workspace/libraries"
	materialized_features "github.com/databricks/cli/cmd/workspace/materialized-features"
	metastores "github.com/databricks/cli/cmd/workspace/metastores"
	model_registry "github.com/databricks/cli/cmd/workspace/model-registry"
	model_versions "github.com/databricks/cli/cmd/workspace/model-versions"
	notification_destinations "github.com/databricks/cli/cmd/workspace/notification-destinations"
	online_tables "github.com/databricks/cli/cmd/workspace/online-tables"
	permission_migration "github.com/databricks/cli/cmd/workspace/permission-migration"
	permissions "github.com/databricks/cli/cmd/workspace/permissions"
	pipelines "github.com/databricks/cli/cmd/workspace/pipelines"
	policies "github.com/databricks/cli/cmd/workspace/policies"
	policy_compliance_for_clusters "github.com/databricks/cli/cmd/workspace/policy-compliance-for-clusters"
	policy_compliance_for_jobs "github.com/databricks/cli/cmd/workspace/policy-compliance-for-jobs"
	policy_families "github.com/databricks/cli/cmd/workspace/policy-families"
	postgres "github.com/databricks/cli/cmd/workspace/postgres"
	provider_exchange_filters "github.com/databricks/cli/cmd/workspace/provider-exchange-filters"
	provider_exchanges "github.com/databricks/cli/cmd/workspace/provider-exchanges"
	provider_files "github.com/databricks/cli/cmd/workspace/provider-files"
	provider_listings "github.com/databricks/cli/cmd/workspace/provider-listings"
	provider_personalization_requests "github.com/databricks/cli/cmd/workspace/provider-personalization-requests"
	provider_provider_analytics_dashboards "github.com/databricks/cli/cmd/workspace/provider-provider-analytics-dashboards"
	provider_providers "github.com/databricks/cli/cmd/workspace/provider-providers"
	providers "github.com/databricks/cli/cmd/workspace/providers"
	quality_monitor_v2 "github.com/databricks/cli/cmd/workspace/quality-monitor-v2"
	quality_monitors "github.com/databricks/cli/cmd/workspace/quality-monitors"
	queries "github.com/databricks/cli/cmd/workspace/queries"
	queries_legacy "github.com/databricks/cli/cmd/workspace/queries-legacy"
	query_history "github.com/databricks/cli/cmd/workspace/query-history"
	query_visualizations "github.com/databricks/cli/cmd/workspace/query-visualizations"
	query_visualizations_legacy "github.com/databricks/cli/cmd/workspace/query-visualizations-legacy"
	recipient_activation "github.com/databricks/cli/cmd/workspace/recipient-activation"
	recipient_federation_policies "github.com/databricks/cli/cmd/workspace/recipient-federation-policies"
	recipients "github.com/databricks/cli/cmd/workspace/recipients"
	redash_config "github.com/databricks/cli/cmd/workspace/redash-config"
	registered_models "github.com/databricks/cli/cmd/workspace/registered-models"
	repos "github.com/databricks/cli/cmd/workspace/repos"
	resource_quotas "github.com/databricks/cli/cmd/workspace/resource-quotas"
	rfa "github.com/databricks/cli/cmd/workspace/rfa"
	sandbox "github.com/databricks/cli/cmd/workspace/sandbox"
	schemas "github.com/databricks/cli/cmd/workspace/schemas"
	secrets "github.com/databricks/cli/cmd/workspace/secrets"
	secrets_uc "github.com/databricks/cli/cmd/workspace/secrets-uc"
	serverless_compute_service "github.com/databricks/cli/cmd/workspace/serverless-compute-service"
	service_principal_secrets_proxy "github.com/databricks/cli/cmd/workspace/service-principal-secrets-proxy"
	service_principals_v2 "github.com/databricks/cli/cmd/workspace/service-principals-v2"
	serving_endpoints "github.com/databricks/cli/cmd/workspace/serving-endpoints"
	settings "github.com/databricks/cli/cmd/workspace/settings"
	shares "github.com/databricks/cli/cmd/workspace/shares"
	software_artifacts "github.com/databricks/cli/cmd/workspace/software-artifacts"
	storage_credentials "github.com/databricks/cli/cmd/workspace/storage-credentials"
	supervisor_agents "github.com/databricks/cli/cmd/workspace/supervisor-agents"
	system_schemas "github.com/databricks/cli/cmd/workspace/system-schemas"
	table_constraints "github.com/databricks/cli/cmd/workspace/table-constraints"
	tables "github.com/databricks/cli/cmd/workspace/tables"
	tag_policies "github.com/databricks/cli/cmd/workspace/tag-policies"
	temporary_path_credentials "github.com/databricks/cli/cmd/workspace/temporary-path-credentials"
	temporary_table_credentials "github.com/databricks/cli/cmd/workspace/temporary-table-credentials"
	temporary_volume_credentials "github.com/databricks/cli/cmd/workspace/temporary-volume-credentials"
	token_management "github.com/databricks/cli/cmd/workspace/token-management"
	tokens "github.com/databricks/cli/cmd/workspace/tokens"
	users_v2 "github.com/databricks/cli/cmd/workspace/users-v2"
	vector_search_endpoints "github.com/databricks/cli/cmd/workspace/vector-search-endpoints"
	vector_search_indexes "github.com/databricks/cli/cmd/workspace/vector-search-indexes"
	volumes "github.com/databricks/cli/cmd/workspace/volumes"
	warehouses "github.com/databricks/cli/cmd/workspace/warehouses"
	workspace "github.com/databricks/cli/cmd/workspace/workspace"
	workspace_bindings "github.com/databricks/cli/cmd/workspace/workspace-bindings"
	workspace_conf "github.com/databricks/cli/cmd/workspace/workspace-conf"
	workspace_entity_tag_assignments "github.com/databricks/cli/cmd/workspace/workspace-entity-tag-assignments"
	workspace_iam_v2 "github.com/databricks/cli/cmd/workspace/workspace-iam-v2"
	workspace_settings_v2 "github.com/databricks/cli/cmd/workspace/workspace-settings-v2"

	groups "github.com/databricks/cli/cmd/workspace/groups"
	service_principals "github.com/databricks/cli/cmd/workspace/service-principals"
	users "github.com/databricks/cli/cmd/workspace/users"
)

// All returns the specs of the workspace commands.
func All() []root.CommandSpec {
	var out []root.CommandSpec

	out = append(out, access_control.Spec)
	out = append(out, agent_bricks.Spec)
	out = append(out, agent_kit.Spec)
	out = append(out, ai_functions.Spec)
	out = append(out, ai_gateway.Spec)
	out = append(out, ai_search.Spec)
	out = append(out, alerts.Spec)
	out = append(out, alerts_legacy.Spec)
	out = append(out, alerts_v2.Spec)
	out = append(out, apps.Spec)
	out = append(out, apps_settings.Spec)
	out = append(out, artifact_allowlists.Spec)
	out = append(out, bundle_deployments.Spec)
	out = append(out, catalogs.Spec)
	out = append(out, clean_room_asset_revisions.Spec)
	out = append(out, clean_room_assets.Spec)
	out = append(out, clean_room_auto_approval_rules.Spec)
	out = append(out, clean_room_task_runs.Spec)
	out = append(out, clean_rooms.Spec)
	out = append(out, cluster_policies.Spec)
	out = append(out, clusters.Spec)
	out = append(out, compute_migrations.Spec)
	out = append(out, connections.Spec)
	out = append(out, consumer_fulfillments.Spec)
	out = append(out, consumer_installations.Spec)
	out = append(out, consumer_listings.Spec)
	out = append(out, consumer_personalization_requests.Spec)
	out = append(out, consumer_providers.Spec)
	out = append(out, credentials.Spec)
	out = append(out, credentials_manager.Spec)
	out = append(out, current_user.Spec)
	out = append(out, dashboard_widgets.Spec)
	out = append(out, dashboards.Spec)
	out = append(out, data_classification.Spec)
	out = append(out, data_quality.Spec)
	out = append(out, data_sources.Spec)
	out = append(out, database.Spec)
	out = append(out, domains.Spec)
	out = append(out, entity_tag_assignments.Spec)
	out = append(out, environments.Spec)
	out = append(out, experiments.Spec)
	out = append(out, external_lineage.Spec)
	out = append(out, external_locations.Spec)
	out = append(out, external_metadata.Spec)
	out = append(out, feature_engineering.Spec)
	out = append(out, feature_store.Spec)
	out = append(out, forecasting.Spec)
	out = append(out, functions.Spec)
	out = append(out, genie.Spec)
	out = append(out, genie_agents.Spec)
	out = append(out, git_credentials.Spec)
	out = append(out, global_init_scripts.Spec)
	out = append(out, grants.Spec)
	out = append(out, groups_v2.Spec)
	out = append(out, instance_pools.Spec)
	out = append(out, instance_profiles.Spec)
	out = append(out, ip_access_lists.Spec)
	out = append(out, jobs.Spec)
	out = append(out, knowledge_assistants.Spec)
	out = append(out, lakeview.Spec)
	out = append(out, lakeview_embedded.Spec)
	out = append(out, lakewatch.Spec)
	out = append(out, libraries.Spec)
	out = append(out, materialized_features.Spec)
	out = append(out, metastores.Spec)
	out = append(out, model_registry.Spec)
	out = append(out, model_versions.Spec)
	out = append(out, notification_destinations.Spec)
	out = append(out, online_tables.Spec)
	out = append(out, permission_migration.Spec)
	out = append(out, permissions.Spec)
	out = append(out, pipelines.Spec)
	out = append(out, policies.Spec)
	out = append(out, policy_compliance_for_clusters.Spec)
	out = append(out, policy_compliance_for_jobs.Spec)
	out = append(out, policy_families.Spec)
	out = append(out, postgres.Spec)
	out = append(out, provider_exchange_filters.Spec)
	out = append(out, provider_exchanges.Spec)
	out = append(out, provider_files.Spec)
	out = append(out, provider_listings.Spec)
	out = append(out, provider_personalization_requests.Spec)
	out = append(out, provider_provider_analytics_dashboards.Spec)
	out = append(out, provider_providers.Spec)
	out = append(out, providers.Spec)
	out = append(out, quality_monitor_v2.Spec)
	out = append(out, quality_monitors.Spec)
	out = append(out, queries.Spec)
	out = append(out, queries_legacy.Spec)
	out = append(out, query_history.Spec)
	out = append(out, query_visualizations.Spec)
	out = append(out, query_visualizations_legacy.Spec)
	out = append(out, recipient_activation.Spec)
	out = append(out, recipient_federation_policies.Spec)
	out = append(out, recipients.Spec)
	out = append(out, redash_config.Spec)
	out = append(out, registered_models.Spec)
	out = append(out, repos.Spec)
	out = append(out, resource_quotas.Spec)
	out = append(out, rfa.Spec)
	out = append(out, sandbox.Spec)
	out = append(out, schemas.Spec)
	out = append(out, secrets.Spec)
	out = append(out, secrets_uc.Spec)
	out = append(out, serverless_compute_service.Spec)
	out = append(out, service_principal_secrets_proxy.Spec)
	out = append(out, service_principals_v2.Spec)
	out = append(out, serving_endpoints.Spec)
	out = append(out, settings.Spec)
	out = append(out, shares.Spec)
	out = append(out, software_artifacts.Spec)
	out = append(out, storage_credentials.Spec)
	out = append(out, supervisor_agents.Spec)
	out = append(out, system_schemas.Spec)
	out = append(out, table_constraints.Spec)
	out = append(out, tables.Spec)
	out = append(out, tag_policies.Spec)
	out = append(out, temporary_path_credentials.Spec)
	out = append(out, temporary_table_credentials.Spec)
	out = append(out, temporary_volume_credentials.Spec)
	out = append(out, token_management.Spec)
	out = append(out, tokens.Spec)
	out = append(out, users_v2.Spec)
	out = append(out, vector_search_endpoints.Spec)
	out = append(out, vector_search_indexes.Spec)
	out = append(out, volumes.Spec)
	out = append(out, warehouses.Spec)
	out = append(out, workspace.Spec)
	out = append(out, workspace_bindings.Spec)
	out = append(out, workspace_conf.Spec)
	out = append(out, workspace_entity_tag_assignments.Spec)
	out = append(out, workspace_iam_v2.Spec)
	out = append(out, workspace_settings_v2.Spec)

	out = append(out, groups.Spec)
	out = append(out, service_principals.Spec)
	out = append(out, users.Spec)

	return out
}

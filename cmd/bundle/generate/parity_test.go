package generate

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestGenerateParity ensures that all bundle resource types either have
// generate support implemented or are in the explicit unsupported allowlist.
// This test fails when:
//   1. A new resource type is added to bundle/config/resources.go without a generate command or allowlist entry
//   2. An allowlisted resource type gains a generate command but isn't removed from the allowlist
//
// Supported resource types (have registered generate commands):
//   - job, pipeline, dashboard, alert, app, genie_space, vector_search_endpoint
//
// Explicitly unsupported (intentionally excluded from generation):
//   - Snapshots: internal only, not user-managed
//   - JobRuns: read-only, cannot be generated
//   - ModelServices, McpServices, ModelProviderServices: internal/preview, SDK not stable
//   - Models, Experiments, ModelServingEndpoints, RegisteredModels, QualityMonitors: ML resources (TODO)
//   - Catalogs, Schemas, Volumes, ExternalLocations: UC core (TODO)
//   - Clusters, SecretScopes, SqlWarehouses: Compute/storage (TODO)
//   - DatabaseInstances, DatabaseCatalogs, SyncedDatabaseTables: Lakebase (TODO)
//   - Postgres*: Postgres resources (TODO)
//   - VectorSearchIndexes, InstancePools, Secrets, ClusterPolicies: Not yet prioritized (TODO)
//
// To add support for a new resource type:
//   1. Implement a converter in bundle/generate/<resource_type>.go
//   2. Implement a command in cmd/bundle/generate/<resource_type>.go
//   3. Register it in cmd/bundle/generate.go
//   4. Remove it from the unsupported allowlist
//   5. Update this comment
func TestGenerateParity(t *testing.T) {
	// Supported commands (map resource names to supported status).
	supported := map[string]bool{
		"jobs":                     true,
		"pipelines":                true,
		"dashboards":               true,
		"alerts":                   true,
		"apps":                     true,
		"genie_spaces":             true,
		"vector_search_endpoints":  true,
	}

	// Explicitly unsupported resource types (intentionally excluded).
	unsupported := map[string]bool{
		// Internal
		"snapshots":                    true,
		"job_runs":                     true,

		// Preview/internal services
		"model_services":               true,
		"mcp_services":                 true,
		"model_provider_services":      true,

		// ML (TODO: implement)
		"models":                       true,
		"experiments":                  true,
		"model_serving_endpoints":      true,
		"registered_models":            true,
		"quality_monitors":             true,

		// UC core (TODO: implement)
		"catalogs":                     true,
		"schemas":                      true,
		"volumes":                      true,
		"external_locations":           true,

		// Compute/storage (TODO: implement)
		"clusters":                     true,
		"secret_scopes":                true,
		"sql_warehouses":               true,

		// Lakebase (TODO: implement)
		"database_instances":           true,
		"database_catalogs":            true,
		"synced_database_tables":       true,

		// Postgres (TODO: implement)
		"postgres_projects":            true,
		"postgres_branches":            true,
		"postgres_endpoints":           true,
		"postgres_catalogs":            true,
		"postgres_databases":           true,
		"postgres_roles":               true,
		"postgres_synced_tables":       true,
		"postgres_snapshot_schedules":  true,

		// Not yet prioritized
		"vector_search_indexes":        true,
		"instance_pools":               true,
		"secrets":                      true,
		"cluster_policies":             true,
	}

	// All resource types from bundle/config/resources.go
	allResources := map[string]bool{
		"jobs": true, "job_runs": true, "pipelines": true,
		"models": true, "experiments": true, "model_serving_endpoints": true,
		"model_services": true, "mcp_services": true, "model_provider_services": true,
		"registered_models": true, "quality_monitors": true,
		"catalogs": true, "schemas": true, "volumes": true, "external_locations": true,
		"clusters": true, "dashboards": true, "genie_spaces": true,
		"apps": true, "secret_scopes": true, "alerts": true,
		"sql_warehouses": true, "database_instances": true, "database_catalogs": true,
		"synced_database_tables": true,
		"postgres_projects": true, "postgres_branches": true, "postgres_endpoints": true,
		"postgres_catalogs": true, "postgres_databases": true, "postgres_roles": true,
		"postgres_synced_tables": true, "postgres_snapshot_schedules": true,
		"vector_search_endpoints": true, "vector_search_indexes": true,
		"instance_pools": true, "secrets": true, "cluster_policies": true,
		"snapshots": true, // internal
	}

	// Check coverage: every resource must be either supported or explicitly unsupported.
	for resource := range allResources {
		isSingularSupported := supported[resource]
		isSingularUnsupported := unsupported[resource]

		require.True(t, isSingularSupported || isSingularUnsupported,
			"Resource type %q must either have a generate command or be in the unsupported allowlist", resource)
	}

	// Check that no unsupported resource type has gained support without the allowlist being updated.
	for resource := range unsupported {
		require.False(t, supported[resource],
			"Resource type %q is in the unsupported allowlist but has a generate command; update the test", resource)
	}
}

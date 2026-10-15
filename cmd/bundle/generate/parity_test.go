package generate

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestGenerateParity documents which bundle resource types have generate support.
// This test should be updated whenever:
// 1. A new resource type is added to bundle/config/resources.go
// 2. A new generate command is implemented
// 3. Generate support is intentionally excluded for a resource type
//
// Supported resource types for generation:
//   - job: command "job" (basic jobs only; spark_python_task / spark_sql_task supported)
//   - pipeline: command "pipeline"
//   - dashboard: command "dashboard"
//   - alert: command "alert"
//   - app: command "app"
//   - genie_space: command "genie_space" (internal, hidden)
//   - vector_search_endpoint: command "vector_search_endpoint" (internal, hidden)
//
// Resource types NOT yet supported for generation:
//   - models, experiments, model_serving_endpoints, registered_models, quality_monitors
//   - catalogs, schemas, volumes, external_locations
//   - clusters, secret_scopes, sql_warehouses
//   - database_instances, database_catalogs, synced_database_tables
//   - postgres_* (projects, branches, endpoints, catalogs, databases, roles, synced_tables, snapshot_schedules)
//   - instance_pools, secrets, cluster_policies
//
// To add support for a new resource type:
//   1. Implement a converter function in bundle/generate/<resource_type>.go
//   2. Implement a command in cmd/bundle/generate/<resource_type>.go
//   3. Register the command in cmd/bundle/generate.go
//   4. Update this comment and test with the new resource type
func TestGenerateParity(t *testing.T) {
	// This is a documentation test; it passes trivially but serves as a
	// commit point for tracking generate support. Keep the comment above
	// updated whenever resource types or generators change.
	require.True(t, true)
}

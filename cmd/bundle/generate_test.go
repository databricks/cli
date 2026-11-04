package bundle

import (
	"maps"
	"slices"
	"testing"

	"github.com/databricks/cli/bundle/config"
	"github.com/stretchr/testify/assert"
)

// generateUnsupported lists the resource groups without a generate command.
// Remove a group once its command lands. A group missing from both this list and
// the command table fails the test, so every new resource type gets a decision.
var generateUnsupported = []string{
	"catalogs",
	"cluster_policies",
	"clusters",
	"data_quality_monitors",
	"database_catalogs",
	"database_instances",
	"experiments",
	"external_locations",
	"features",
	"instance_pools",
	"internal_immutable_snapshots",
	"job_runs",
	"materialization_sets",
	"mcp_services",
	"model_provider_services",
	"model_services",
	"model_serving_endpoints",
	"models",
	"postgres_branches",
	"postgres_catalogs",
	"postgres_databases",
	"postgres_endpoints",
	"postgres_projects",
	"postgres_roles",
	"postgres_snapshot_schedules",
	"postgres_synced_tables",
	"quality_monitors",
	"registered_models",
	"schemas",
	"secret_scopes",
	"secrets",
	"sql_warehouses",
	"synced_database_tables",
	"vector_search_endpoints",
	"vector_search_indexes",
	"volumes",
}

func TestGenerateCommandCoversAllResourceGroups(t *testing.T) {
	// Generate subcommand name -> resource group it writes.
	supported := map[string]string{
		"alert":       "alerts",
		"app":         "apps",
		"dashboard":   "dashboards",
		"genie-space": "genie_spaces",
		"job":         "jobs",
		"pipeline":    "pipelines",
	}

	var names []string
	for _, sub := range newGenerateCommand().Commands() {
		names = append(names, sub.Name())
	}
	assert.ElementsMatch(t, slices.Collect(maps.Keys(supported)), names)

	groups := config.SupportedResources()
	covered := slices.Collect(maps.Values(supported))
	for _, group := range generateUnsupported {
		assert.Contains(t, groups, group, "%q is not a bundle resource group; drop it from generateUnsupported", group)
		assert.NotContains(t, covered, group, "%q has a generate command; drop it from generateUnsupported", group)
	}
	for group := range groups {
		if slices.Contains(generateUnsupported, group) {
			continue
		}
		assert.Contains(t, covered, group, "%q has no generate command; add one or list it in generateUnsupported", group)
	}
}

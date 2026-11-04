package bundle

import (
	"slices"
	"strings"
	"testing"

	"github.com/databricks/cli/bundle/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// generateUnsupported lists the resource groups without a generate subcommand.
// Remove a group once its command lands. A group that is in neither this list
// nor the command tree fails the test, so every new resource type gets a decision.
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
	groups := config.SupportedResources()

	// Subcommands are named after the resource type's singular name with dashes.
	byCommand := make(map[string]string, len(groups))
	for group, d := range groups {
		byCommand[strings.ReplaceAll(d.SingularName, "_", "-")] = group
	}

	covered := make(map[string]bool)
	for _, sub := range newGenerateCommand().Commands() {
		group, ok := byCommand[sub.Name()]
		require.True(t, ok, "generate subcommand %q does not match the singular name of any resource type", sub.Name())
		covered[group] = true
	}

	for _, group := range generateUnsupported {
		assert.Contains(t, groups, group, "%q is not a resource group; drop it from generateUnsupported", group)
		assert.False(t, covered[group], "%q has a generate subcommand; drop it from generateUnsupported", group)
	}
	for group := range groups {
		if slices.Contains(generateUnsupported, group) {
			continue
		}
		assert.True(t, covered[group], "%q has no generate subcommand; add one or list it in generateUnsupported", group)
	}
}

package configsync

import (
	"slices"
	"strings"

	"github.com/databricks/cli/bundle/direct/dresources"
	"github.com/databricks/cli/libs/structs/structpath"
)

// shouldSkipForSync reports whether a remote change at path should be excluded
// from config sync, based on the resource lifecycle metadata (resources.yml and
// resources.generated.yml in bundle/direct/dresources).
//
// The metadata is matched directly instead of relying on the plan's per-field
// actions: the plan answers "what will deploy do", which is not the same as
// "does this field belong in configuration". For example, the plan re-promotes
// dashboard etag drift to Update via ResourceDashboard.OverrideChangeDesc (for
// modified-remotely detection), yet etag must never be written to YAML.
//
// Fields under ignore_remote_changes never belong in configuration (output-only,
// etag-based, input-only), so they are skipped regardless of value. Fields under
// backend_defaults are skipped only when absent from the config (hasConfigValue
// is false) and the remote value matches the rule.
func shouldSkipForSync(resourceType string, path *structpath.PathNode, value any, hasConfigValue bool) bool {
	cfg := dresources.GetResourceConfig(resourceType)
	generatedCfg := dresources.GetGeneratedResourceConfig(resourceType)

	if _, ok := dresources.FindMatchingRule(path, cfg.IgnoreRemoteChanges); ok {
		return true
	}
	if _, ok := dresources.FindMatchingRule(path, generatedCfg.IgnoreRemoteChanges); ok {
		return true
	}

	if hasConfigValue {
		return false
	}

	return cfg.MatchesBackendDefault(path, value) || generatedCfg.MatchesBackendDefault(path, value)
}

// fieldsIgnoredForSync defines fields set by the CLI that cannot be specified
// in databricks.yml and should be excluded from config-remote-sync results.
// These are sync-specific rules: the deploy plan has different drift semantics
// for them, so they do not belong in resources.yml.
var fieldsIgnoredForSync = map[string][]*structpath.PatternNode{
	"jobs": {
		structpath.MustParsePattern("edit_mode"),
	},
}

// fieldsKeptForSync lists fields that nested entity filtering (filterEntityDefaults)
// must keep even though they match a backend_defaults rule. The rules are asymmetric
// between plan and sync: for the plan, node_type_id is computed — policy-backed
// clusters materialize the policy-resolved value remotely while config legitimately
// omits it, so leaf-level drift must be skipped. For sync, a remotely added cluster's
// explicit node_type_id is required configuration: stripping it would make the synced
// YAML undeployable (cluster creation fails with 400 "Unknown node type id" unless a
// pool or policy provides the node type).
//
// Keeping it unconditionally cannot produce a node_type_id/instance_pool_id conflict:
// the Jobs API rejects requests carrying both ("The field 'node_type_id' cannot be
// supplied when an instance pool ID is provided") and jobs/get does not materialize
// node_type_id for pool-backed clusters, so a remote cluster never has both fields.
var fieldsKeptForSync = map[string][]*structpath.PatternNode{
	"jobs": {
		structpath.MustParsePattern("tasks[*].new_cluster.node_type_id"),
		structpath.MustParsePattern("job_clusters[*].new_cluster.node_type_id"),
	},
}

// isKeptForSync reports whether a field of a remotely added entity must survive
// filterEntityDefaults even though it matches the lifecycle metadata.
func isKeptForSync(resourceType string, path *structpath.PathNode) bool {
	return slices.ContainsFunc(fieldsKeptForSync[resourceType], path.HasPatternPrefix)
}

func isIgnoredForSync(resourceType string, path *structpath.PathNode) bool {
	return slices.ContainsFunc(fieldsIgnoredForSync[resourceType], path.HasPatternPrefix)
}

type resetRule struct {
	field *structpath.PatternNode
	value any
}

// resetValues contains all values that should be used to reset CLI-defaulted fields.
// If CLI-defaulted field is changed on remote and should be disabled (e.g. queueing disabled -> remote field is nil)
// we can't define it in the config as "null" because CLI default will be applied again.
var resetValues = map[string][]resetRule{
	"jobs": {
		{field: structpath.MustParsePattern("queue"), value: map[string]any{"enabled": false}},
	},
}

func resetValueIfNeeded(resourceType string, path *structpath.PathNode, value any) any {
	for _, rule := range resetValues[resourceType] {
		// Exact match, not prefix: a change at a subfield (e.g. queue.enabled)
		// must keep its own value; only a change of the whole field is reset.
		if path.Len() == rule.field.Len() && path.HasPatternPrefix(rule.field) {
			return rule.value
		}
	}
	return value
}

// prefixedNameFields lists resource name fields where the name prefix
// (e.g. "[dev user] ") is applied during deployment and should be stripped
// when syncing remote changes back to config.
var prefixedNameFields = map[string][]*structpath.PatternNode{
	"jobs":       {structpath.MustParsePattern("name")},
	"pipelines":  {structpath.MustParsePattern("name")},
	"dashboards": {structpath.MustParsePattern("display_name")},
}

// stripNamePrefix strips the configured name prefix from name field values
// so that the raw (unprefixed) name is written back to the config YAML.
func stripNamePrefix(resourceType string, path *structpath.PathNode, value any, prefix string) any {
	if prefix == "" {
		return value
	}

	s, ok := value.(string)
	if !ok {
		return value
	}

	if slices.ContainsFunc(prefixedNameFields[resourceType], path.HasPatternPrefix) {
		return strings.TrimPrefix(s, prefix)
	}

	return value
}

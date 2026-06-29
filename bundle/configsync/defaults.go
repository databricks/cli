package configsync

import (
	"reflect"
	"strings"
)

type (
	skipAlways           struct{}
	skipIfZeroOrNil      struct{}
	skipIfEmptyOrDefault struct {
		defaults map[string]any
	}
	// skipBackendDefault skips a field, regardless of its remote value, when it is
	// absent from config. Used for fields the backend or a cluster policy may fill
	// in: their remote-only value must not be synced into config. Fields the user
	// does manage (present in config) still sync normally.
	skipBackendDefault struct{}
)

var (
	alwaysSkip              = skipAlways{}
	zeroOrNil               = skipIfZeroOrNil{}
	emptyEmailNotifications = skipIfEmptyOrDefault{defaults: map[string]any{"no_alert_for_skipped_runs": false}}
	backendDefault          = skipBackendDefault{}
)

// serverSideDefaults contains all hardcoded server-side defaults.
// This is a temporary solution until the bundle plan issue is resolved.
// Fields mapped to alwaysSkip are always considered defaults regardless of value.
// Other fields are compared using reflect.DeepEqual.
var serverSideDefaults = map[string]any{
	// Job-level fields
	"resources.jobs.*.timeout_seconds":       zeroOrNil,
	"resources.jobs.*.email_notifications":   emptyEmailNotifications,
	"resources.jobs.*.webhook_notifications": map[string]any{},
	"resources.jobs.*.edit_mode":             alwaysSkip, // set by CLI
	"resources.jobs.*.performance_target":    "PERFORMANCE_OPTIMIZED",

	// Task-level fields
	"resources.jobs.*.tasks[*].run_if":                     "ALL_SUCCESS",
	"resources.jobs.*.tasks[*].disabled":                   false,
	"resources.jobs.*.tasks[*].timeout_seconds":            zeroOrNil,
	"resources.jobs.*.tasks[*].notebook_task.source":       "WORKSPACE",
	"resources.jobs.*.tasks[*].email_notifications":        emptyEmailNotifications,
	"resources.jobs.*.tasks[*].webhook_notifications":      map[string]any{},
	"resources.jobs.*.tasks[*].pipeline_task.full_refresh": false,

	"resources.jobs.*.tasks[*].for_each_task.task.run_if":                "ALL_SUCCESS",
	"resources.jobs.*.tasks[*].for_each_task.task.disabled":              false,
	"resources.jobs.*.tasks[*].for_each_task.task.timeout_seconds":       zeroOrNil,
	"resources.jobs.*.tasks[*].for_each_task.task.notebook_task.source":  "WORKSPACE",
	"resources.jobs.*.tasks[*].for_each_task.task.email_notifications":   emptyEmailNotifications,
	"resources.jobs.*.tasks[*].for_each_task.task.webhook_notifications": map[string]any{},

	// Cluster fields (tasks)
	"resources.jobs.*.tasks[*].new_cluster.aws_attributes":      alwaysSkip,
	"resources.jobs.*.tasks[*].new_cluster.azure_attributes":    alwaysSkip,
	"resources.jobs.*.tasks[*].new_cluster.gcp_attributes":      alwaysSkip,
	"resources.jobs.*.tasks[*].new_cluster.data_security_mode":  "SINGLE_USER", // TODO this field is computed on some workspaces in integration tests, check why and if we can skip it
	"resources.jobs.*.tasks[*].new_cluster.enable_elastic_disk": alwaysSkip,    // deprecated field
	"resources.jobs.*.tasks[*].new_cluster.single_user_name":    alwaysSkip,
	// custom_tags and cluster_log_conf are commonly injected by cluster policies
	// when the user omits them, so they exist only remotely. Syncing them back leaks
	// one environment's policy values into (often shared) config and breaks deploys in
	// other environments. TODO: move to backend_defaults in resources.yml once
	// configsync filtering is migrated to the direct engine lifecycle metadata.
	"resources.jobs.*.tasks[*].new_cluster.custom_tags":      backendDefault,
	"resources.jobs.*.tasks[*].new_cluster.cluster_log_conf": backendDefault,

	// Cluster fields (job_clusters)
	"resources.jobs.*.job_clusters[*].new_cluster.aws_attributes":      alwaysSkip,
	"resources.jobs.*.job_clusters[*].new_cluster.azure_attributes":    alwaysSkip,
	"resources.jobs.*.job_clusters[*].new_cluster.gcp_attributes":      alwaysSkip,
	"resources.jobs.*.job_clusters[*].new_cluster.data_security_mode":  "SINGLE_USER", // TODO this field is computed on some workspaces in integration tests, check why and if we can skip it
	"resources.jobs.*.job_clusters[*].new_cluster.enable_elastic_disk": alwaysSkip,    // deprecated field
	"resources.jobs.*.job_clusters[*].new_cluster.single_user_name":    alwaysSkip,
	"resources.jobs.*.job_clusters[*].new_cluster.custom_tags":         backendDefault, // see tasks[*].new_cluster.custom_tags
	"resources.jobs.*.job_clusters[*].new_cluster.cluster_log_conf":    backendDefault, // see tasks[*].new_cluster.cluster_log_conf

	// Standalone cluster fields
	"resources.clusters.*.aws_attributes":      alwaysSkip,
	"resources.clusters.*.azure_attributes":    alwaysSkip,
	"resources.clusters.*.gcp_attributes":      alwaysSkip,
	"resources.clusters.*.data_security_mode":  "SINGLE_USER",
	"resources.clusters.*.driver_node_type_id": alwaysSkip,
	"resources.clusters.*.enable_elastic_disk": alwaysSkip,
	"resources.clusters.*.single_user_name":    alwaysSkip,
	"resources.clusters.*.custom_tags":         backendDefault, // see jobs.*.tasks[*].new_cluster.custom_tags
	"resources.clusters.*.cluster_log_conf":    backendDefault, // see jobs.*.tasks[*].new_cluster.cluster_log_conf

	// Experiment fields
	"resources.experiments.*.artifact_location": alwaysSkip,

	// Registered model fields
	"resources.registered_models.*.full_name":        alwaysSkip,
	"resources.registered_models.*.metastore_id":     alwaysSkip,
	"resources.registered_models.*.owner":            alwaysSkip,
	"resources.registered_models.*.storage_location": alwaysSkip,

	// Volume fields
	"resources.volumes.*.storage_location": alwaysSkip,

	// SQL warehouse fields
	"resources.sql_warehouses.*.creator_name":     alwaysSkip,
	"resources.sql_warehouses.*.min_num_clusters": int64(1),
	"resources.sql_warehouses.*.warehouse_type":   "CLASSIC",

	// Terraform defaults
	"resources.jobs.*.run_as": alwaysSkip,

	// Pipeline fields
	"resources.pipelines.*.storage":    alwaysSkip,
	"resources.pipelines.*.continuous": false,

	// Dashboard fields
	// etag is output-only and never present in config. The plan re-promotes etag
	// drift to Update via ResourceDashboard.OverrideChangeDesc (needed for deploy's
	// modified-remotely detection), so configsync cannot rely on the plan's Skip
	// action and must exclude the field explicitly.
	"resources.dashboards.*.etag": alwaysSkip,
}

// shouldSkipField checks if a field should be skipped in change detection.
// When hasConfigValue is true (field is set in config or saved state), only
// "always skip" fields are skipped. Backend defaults are only skipped when the
// field is not in config/state, matching the behavior of shouldSkipBackendDefault
// in the direct deployment engine.
func shouldSkipField(path string, value any, hasConfigValue bool) bool {
	for pattern, expected := range serverSideDefaults {
		if matchPattern(pattern, path) {
			if _, ok := expected.(skipAlways); ok {
				return true
			}
			if hasConfigValue {
				return false
			}
			if _, ok := expected.(skipBackendDefault); ok {
				return true
			}
			if _, ok := expected.(skipIfZeroOrNil); ok {
				return value == nil || value == int64(0)
			}
			if marker, ok := expected.(skipIfEmptyOrDefault); ok {
				m, ok := value.(map[string]any)
				if !ok {
					return false
				}
				if len(m) == 0 {
					return true
				}
				return reflect.DeepEqual(m, marker.defaults)
			}
			return reflect.DeepEqual(value, expected)
		}
	}
	return false
}

func matchPattern(pattern, path string) bool {
	patternParts := strings.Split(pattern, ".")
	pathParts := strings.Split(path, ".")
	return matchParts(patternParts, pathParts)
}

func matchParts(patternParts, pathParts []string) bool {
	if len(patternParts) == 0 && len(pathParts) == 0 {
		return true
	}
	if len(patternParts) == 0 || len(pathParts) == 0 {
		return false
	}

	patternPart := patternParts[0]
	pathPart := pathParts[0]

	if patternPart == "*" {
		return matchParts(patternParts[1:], pathParts[1:])
	}

	if strings.Contains(patternPart, "[*]") {
		prefix, _, _ := strings.Cut(patternPart, "[*]")

		if strings.HasPrefix(pathPart, prefix) && strings.Contains(pathPart, "[") {
			return matchParts(patternParts[1:], pathParts[1:])
		}
		return false
	}

	if patternPart == pathPart {
		return matchParts(patternParts[1:], pathParts[1:])
	}

	return false
}

// resetValues contains all values that should be used to reset CLI-defaulted fields.
// If CLI-defaulted field is changed on remote and should be disabled (e.g. queueing disabled -> remote field is nil)
// we can't define it in the config as "null" because CLI default will be applied again.
var resetValues = map[string]any{
	"resources.jobs.*.queue": map[string]any{
		"enabled": false,
	},
}

func resetValueIfNeeded(path string, value any) any {
	for pattern, expected := range resetValues {
		if matchPattern(pattern, path) {
			return expected
		}
	}
	return value
}

// prefixedNameFields lists resource name field patterns where the name prefix
// (e.g. "[dev user] ") is applied during deployment and should be stripped
// when syncing remote changes back to config.
var prefixedNameFields = []string{
	"resources.jobs.*.name",
	"resources.pipelines.*.name",
	"resources.dashboards.*.display_name",
}

// stripNamePrefix strips the configured name prefix from name field values
// so that the raw (unprefixed) name is written back to the config YAML.
func stripNamePrefix(path string, value any, prefix string) any {
	if prefix == "" {
		return value
	}

	s, ok := value.(string)
	if !ok {
		return value
	}

	for _, pattern := range prefixedNameFields {
		if matchPattern(pattern, path) {
			return strings.TrimPrefix(s, prefix)
		}
	}

	return value
}

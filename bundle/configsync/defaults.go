package configsync

import (
	"reflect"
	"strings"
)

// skipAlways is a marker type for fields that should always be considered server-side defaults
type skipAlways struct{}

var alwaysSkip = skipAlways{}

// serverSideDefaults contains all hardcoded server-side defaults.
// This is a temporary solution until the bundle plan issue is resolved.
// Fields mapped to alwaysSkip are always considered defaults regardless of value.
// Other fields are compared using reflect.DeepEqual.
var serverSideDefaults = map[string]any{
	// Job-level fields
	"timeout_seconds": nil,
	// "usage_policy_id":     alwaysSkip, // computed field
	"edit_mode": alwaysSkip, // set by CLI
	"email_notifications": map[string]any{
		"no_alert_for_skipped_runs": false,
	},
	"performance_target": "PERFORMANCE_OPTIMIZED",

	// Task-level fields
	"tasks[*].run_if":                     "ALL_SUCCESS",
	"tasks[*].disabled":                   false,
	"tasks[*].timeout_seconds":            nil,
	"tasks[*].notebook_task.source":       "WORKSPACE",
	"tasks[*].email_notifications":        map[string]any{"no_alert_for_skipped_runs": false},
	"tasks[*].pipeline_task.full_refresh": false,

	"tasks[*].for_each_task.task.run_if":               "ALL_SUCCESS",
	"tasks[*].for_each_task.task.disabled":             false,
	"tasks[*].for_each_task.task.timeout_seconds":      nil,
	"tasks[*].for_each_task.task.notebook_task.source": "WORKSPACE",
	"tasks[*].for_each_task.task.email_notifications":  map[string]any{"no_alert_for_skipped_runs": false},

	// Cluster fields (tasks)
	"tasks[*].new_cluster.aws_attributes":      alwaysSkip,
	"tasks[*].new_cluster.azure_attributes":    alwaysSkip,
	"tasks[*].new_cluster.gcp_attributes":      alwaysSkip,
	"tasks[*].new_cluster.data_security_mode":  "SINGLE_USER", // TODO this field is computed on some workspaces in integration tests, check why and if we can skip it
	"tasks[*].new_cluster.enable_elastic_disk": alwaysSkip,    // deprecated field

	// Cluster fields (job_clusters)
	"job_clusters[*].new_cluster.aws_attributes":     alwaysSkip,
	"job_clusters[*].new_cluster.azure_attributes":   alwaysSkip,
	"job_clusters[*].new_cluster.gcp_attributes":     alwaysSkip,
	"job_clusters[*].new_cluster.data_security_mode": "SINGLE_USER", // TODO this field is computed on some workspaces in integration tests, check why and if we can skip it

	"job_clusters[*].new_cluster.enable_elastic_disk": alwaysSkip, // deprecated field

	// Terraform defaults
	"run_as": alwaysSkip,

	// Pipeline fields
	"storage":    alwaysSkip,
	"continuous": false,
}

func shouldSkipField(path string, value any) bool {
	for pattern, expected := range serverSideDefaults {
		if matchPattern(pattern, path) {
			if _, ok := expected.(skipAlways); ok {
				return true
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
		prefix := strings.Split(patternPart, "[*]")[0]

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

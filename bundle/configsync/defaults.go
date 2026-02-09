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
)

var (
	alwaysSkip              = skipAlways{}
	zeroOrNil               = skipIfZeroOrNil{}
	emptyEmailNotifications = skipIfEmptyOrDefault{defaults: map[string]any{"no_alert_for_skipped_runs": false}}
)

// serverSideDefaults contains all hardcoded server-side defaults.
// This is a temporary solution until the bundle plan issue is resolved.
// Fields mapped to alwaysSkip are always considered defaults regardless of value.
// Other fields are compared using reflect.DeepEqual.
var serverSideDefaults = map[string]any{
	// Job-level fields
	"timeout_seconds":       zeroOrNil,
	"email_notifications":   emptyEmailNotifications,
	"webhook_notifications": map[string]any{},
	"edit_mode":             alwaysSkip, // set by CLI
	"performance_target":    "PERFORMANCE_OPTIMIZED",

	// Task-level fields
	"tasks[*].run_if":                     "ALL_SUCCESS",
	"tasks[*].disabled":                   false,
	"tasks[*].timeout_seconds":            zeroOrNil,
	"tasks[*].notebook_task.source":       "WORKSPACE",
	"tasks[*].email_notifications":        emptyEmailNotifications,
	"tasks[*].webhook_notifications":      map[string]any{},
	"tasks[*].pipeline_task.full_refresh": false,

	"tasks[*].for_each_task.task.run_if":                "ALL_SUCCESS",
	"tasks[*].for_each_task.task.disabled":              false,
	"tasks[*].for_each_task.task.timeout_seconds":       zeroOrNil,
	"tasks[*].for_each_task.task.notebook_task.source":  "WORKSPACE",
	"tasks[*].for_each_task.task.email_notifications":   emptyEmailNotifications,
	"tasks[*].for_each_task.task.webhook_notifications": map[string]any{},

	// Cluster fields (tasks)
	"tasks[*].new_cluster.data_security_mode": "SINGLE_USER", // TODO this field is computed on some workspaces in integration tests, check why and if we can skip it

	// Cluster fields (job_clusters)
	"job_clusters[*].new_cluster.data_security_mode": "SINGLE_USER", // TODO this field is computed on some workspaces in integration tests, check why and if we can skip it

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

// resetValues contains all values that should be used to reset CLI-defaulted fields.
// If CLI-defaulted field is changed on remote and should be disabled (e.g. queueing disabled -> remote field is nil)
// we can't define it in the config as "null" because CLI default will be applied again.
var resetValues = map[string]any{
	"queue": map[string]any{
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

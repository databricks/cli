package configsync

import (
	"testing"

	"github.com/databricks/cli/libs/dyn"
	"github.com/stretchr/testify/assert"
)

func TestCollectChangeStats(t *testing.T) {
	changes := Changes{
		"resources.jobs.foo": {
			"name":                   {Operation: OperationReplace, Value: "new name"},
			"tasks[0].notebook_task": {Operation: OperationAdd, Value: map[string]any{"base_parameters": map[string]any{"p": "${workspace.file_path}/x"}}},
			"timeout_seconds":        {Operation: OperationRemove},
		},
		"resources.jobs.bar": {
			"name": {Operation: OperationReplace, Value: "other"},
		},
		"resources.dashboards.dash": {
			"etag": {Operation: OperationAdd, Value: "123456"},
		},
	}

	var stats Stats
	stats.CollectChangeStats(changes)

	assert.Equal(t, int64(5), stats.ChangesTotal)
	assert.Equal(t, int64(2), stats.AddCount)
	assert.Equal(t, int64(2), stats.ReplaceCount)
	assert.Equal(t, int64(1), stats.RemoveCount)
	assert.Equal(t, int64(1), stats.RawValuesWithVarSyntax)

	jobs := stats.PerResourceType["jobs"]
	assert.Equal(t, int64(4), jobs.ChangesCount)
	assert.Equal(t, int64(1), jobs.AddCount)
	assert.Equal(t, int64(2), jobs.ReplaceCount)
	assert.Equal(t, int64(1), jobs.RemoveCount)

	dashboards := stats.PerResourceType["dashboards"]
	assert.Equal(t, int64(1), dashboards.ChangesCount)
	assert.Equal(t, int64(1), dashboards.AddCount)
}

func TestResourceTypeFromKey(t *testing.T) {
	assert.Equal(t, "jobs", resourceTypeFromKey("resources.jobs.foo"))
	assert.Equal(t, "dashboards", resourceTypeFromKey("resources.dashboards.a.b"))
	assert.Equal(t, "unknown", resourceTypeFromKey("variables.foo"))
	assert.Equal(t, "unknown", resourceTypeFromKey("resources"))
}

func TestCountVarSyntax(t *testing.T) {
	assert.Equal(t, int64(0), countVarSyntax("plain"))
	assert.Equal(t, int64(1), countVarSyntax("${var.x}"))
	assert.Equal(t, int64(1), countVarSyntax("prefix ${ suffix"))
	assert.Equal(t, int64(2), countVarSyntax(map[string]any{
		"a": "${var.x}",
		"b": []any{"${incomplete", "ok", int64(5)},
	}))
}

func TestRestoreStatsCounters(t *testing.T) {
	resolved := dyn.V(map[string]dyn.Value{
		"variables": dyn.V(map[string]dyn.Value{
			"region": dyn.V(map[string]dyn.Value{"value": dyn.V("us-east-1")}),
			"other":  dyn.V(map[string]dyn.Value{"value": dyn.V("eu-west-1")}),
		}),
	})

	// Original pure ref still matching: restored but not counted (safe path).
	var kept RestoreStats
	result := restoreOriginalRefs("us-east-1", dyn.V("${var.region}"), resolved, &kept)
	assert.Equal(t, "${var.region}", result)
	assert.Equal(t, RestoreStats{}, kept)

	// Pure ref whose value changed to another variable's value: re-targeted.
	var retargeted RestoreStats
	result = restoreOriginalRefs("eu-west-1", dyn.V("${var.region}"), resolved, &retargeted)
	assert.Equal(t, "${var.other}", result)
	assert.Equal(t, RestoreStats{Retargeted: 1}, retargeted)

	// New sequence element leaf restored from a sibling reference.
	siblings := []dyn.Value{
		dyn.V(map[string]dyn.Value{"region": dyn.V("${var.region}")}),
	}
	var fromSiblings RestoreStats
	resultMap := restoreFromSiblings(map[string]any{"region": "us-east-1"}, siblings, resolved, &fromSiblings).(map[string]any)
	assert.Equal(t, "${var.region}", resultMap["region"])
	assert.Equal(t, RestoreStats{FromSiblings: 1}, fromSiblings)

	// Hardcoded value: nothing restored, nothing counted.
	var none RestoreStats
	result = restoreOriginalRefs("hardcoded", dyn.V("hardcoded"), resolved, &none)
	assert.Equal(t, "hardcoded", result)
	assert.Equal(t, RestoreStats{}, none)
}

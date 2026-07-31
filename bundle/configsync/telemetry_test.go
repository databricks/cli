package configsync

import (
	"encoding/json"
	"testing"

	"github.com/databricks/cli/bundle/config/engine"
	"github.com/databricks/cli/bundle/statemgmt"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/telemetry/protos"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCollectChangeStats(t *testing.T) {
	changes := Changes{
		"resources.jobs.foo": {
			"name":                   {Operation: OperationReplace, Value: "new name"},
			"tasks[0].notebook_task": {Operation: OperationAdd, Value: map[string]any{"base_parameters": map[string]any{"p": "x"}}},
			"timeout_seconds":        {Operation: OperationRemove},
		},
		"resources.jobs.bar": {
			"name": {Operation: OperationReplace, Value: "other"},
		},
		"resources.dashboards.dash": {
			"etag": {Operation: OperationAdd, Value: "123456"},
		},
		// pipelines.storage is recreate_on_changes (immutable); also mark it as
		// overwriting a local edit.
		"resources.pipelines.pipe": {
			"storage": {Operation: OperationReplace, Value: "s3://new", LocalEdit: true},
		},
	}

	var stats Stats
	stats.CollectChangeStats(t.Context(), changes)

	assert.Equal(t, int64(6), stats.ChangesTotal)
	assert.Equal(t, int64(2), stats.AddCount)
	assert.Equal(t, int64(3), stats.ReplaceCount)
	assert.Equal(t, int64(1), stats.RemoveCount)
	assert.Equal(t, int64(1), stats.RecreateForcingChanges)
	assert.Equal(t, int64(1), stats.OverwrittenLocalEdits)

	jobs := stats.PerResourceType["jobs"]
	assert.Equal(t, int64(4), jobs.ChangesCount)
	assert.Equal(t, int64(1), jobs.AddCount)
	assert.Equal(t, int64(2), jobs.ReplaceCount)
	assert.Equal(t, int64(1), jobs.RemoveCount)

	dashboards := stats.PerResourceType["dashboards"]
	assert.Equal(t, int64(1), dashboards.ChangesCount)
	assert.Equal(t, int64(1), dashboards.AddCount)
}

func TestIsRecreateForcing(t *testing.T) {
	assert.True(t, isRecreateForcing("pipelines", "storage"))
	assert.False(t, isRecreateForcing("pipelines", "configuration"))
	assert.False(t, isRecreateForcing("jobs", "name"))
	assert.False(t, isRecreateForcing("unknown", "storage"))
}

func TestResourceTypeFromKey(t *testing.T) {
	assert.Equal(t, "jobs", resourceTypeFromKey("resources.jobs.foo"))
	assert.Equal(t, "dashboards", resourceTypeFromKey("resources.dashboards.a.b"))
	assert.Equal(t, "unknown", resourceTypeFromKey("variables.foo"))
	assert.Equal(t, "unknown", resourceTypeFromKey("resources"))
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

func TestRestoreStatsCountersNilSafe(t *testing.T) {
	// Telemetry must never panic the command: counting on a nil pointer is a no-op.
	var s *RestoreStats
	assert.NotPanics(t, func() {
		s.incRetargeted()
		s.incFromSiblings()
	})
}

func TestCollectStateStats(t *testing.T) {
	t.Run("remote state with candidates", func(t *testing.T) {
		s := &Stats{}
		s.CollectStateStats(&statemgmt.StateDesc{
			Serial:    7,
			Lineage:   "f1621b9c-6ccd-481b-854d-40fa4176e68c",
			IsLocal:   false,
			AllStates: []*statemgmt.StateDesc{{}, {}},
		})
		assert.Equal(t, int64(7), s.StateSerial)
		assert.Equal(t, "f1621b9c-6ccd-481b-854d-40fa4176e68c", s.StateLineage)
		assert.Equal(t, "remote", s.StateSource)
		require.NotNil(t, s.StatesAvailableCount)
		assert.Equal(t, int64(2), *s.StatesAvailableCount)
	})

	t.Run("local state wins", func(t *testing.T) {
		s := &Stats{}
		s.CollectStateStats(&statemgmt.StateDesc{Serial: 99, IsLocal: true})
		assert.Equal(t, int64(99), s.StateSerial)
		assert.Equal(t, "local", s.StateSource)
	})

	t.Run("synthesized empty state reports no candidates", func(t *testing.T) {
		// PullResourcesState synthesizes a descriptor with no AllStates when no
		// state file was found; that must be visible as 0, not omitted.
		s := &Stats{}
		s.CollectStateStats(&statemgmt.StateDesc{IsLocal: true})
		require.NotNil(t, s.StatesAvailableCount)
		assert.Equal(t, int64(0), *s.StatesAvailableCount)
		assert.Equal(t, int64(0), s.StateSerial)
		assert.Empty(t, s.StateLineage)
	})

	t.Run("nil descriptor is a no-op", func(t *testing.T) {
		s := &Stats{}
		s.CollectStateStats(nil)
		assert.Empty(t, s.StateSource)
	})
}

// The new state/selector fields must reach the emitted payload, and must be
// omitted (not zero-valued noise) when the run recorded nothing.
func TestLogTelemetryPayloadIncludesStateAndSelectorFields(t *testing.T) {
	s := &Stats{
		Engine:               engine.EngineDirect,
		StateSerial:          99,
		StateLineage:         "f1621b9c-6ccd-481b-854d-40fa4176e68c",
		StateSource:          "local",
		StatesAvailableCount: ptr(int64(2)),
		SelectorCount:        ptr(int64(3)),
		SelectorMatchedCount: ptr(int64(0)),
	}
	payload, err := json.Marshal(protos.BundleConfigRemoteSyncEvent{
		Engine:               string(s.Engine),
		StateSerial:          s.StateSerial,
		StateLineage:         s.StateLineage,
		StateSource:          s.StateSource,
		StatesAvailableCount: s.StatesAvailableCount,
		SelectorCount:        s.SelectorCount,
		SelectorMatchedCount: s.SelectorMatchedCount,
	})
	require.NoError(t, err)

	got := string(payload)
	assert.Contains(t, got, `"state_serial":99`)
	assert.Contains(t, got, `"state_lineage":"f1621b9c-6ccd-481b-854d-40fa4176e68c"`)
	assert.Contains(t, got, `"state_source":"local"`)
	assert.Contains(t, got, `"states_available_count":2`)
	assert.Contains(t, got, `"selector_count":3`)
	// The zero must survive: "no selector matched" is the hard-failure signal.
	assert.Contains(t, got, `"selector_matched_count":0`)

	empty, err := json.Marshal(protos.BundleConfigRemoteSyncEvent{})
	require.NoError(t, err)
	for _, f := range []string{"state_serial", "state_lineage", "state_source", "states_available_count", "selector_count"} {
		assert.NotContains(t, string(empty), f)
	}
}

func ptr[T any](v T) *T { return &v }

package direct

import (
	"bytes"
	"reflect"
	"testing"

	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/bundle/deployplan"
	"github.com/databricks/cli/bundle/direct/dresources"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/dyn/yamlloader"
	"github.com/databricks/cli/libs/structs/structpath"
	"github.com/databricks/cli/libs/structs/structvar"
	"github.com/databricks/databricks-sdk-go/service/jobs"
	"github.com/databricks/databricks-sdk-go/service/pipelines"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDynPathToStructPath(t *testing.T) {
	tests := []struct {
		path     dyn.Path
		expected string
	}{
		{
			path:     dyn.NewPath(dyn.Key("foo"), dyn.Key("bar")),
			expected: "foo.bar",
		},
		{
			path:     dyn.NewPath(dyn.Key("foo"), dyn.Index(1), dyn.Key("bar")),
			expected: "foo[1].bar",
		},
		{
			path:     dyn.NewPath(dyn.Key("configuration"), dyn.Key("europris.swipe.egress_streaming_schema")),
			expected: "configuration['europris.swipe.egress_streaming_schema']",
		},
		{
			path:     dyn.NewPath(dyn.Key("tags"), dyn.Key("it's.here")),
			expected: "tags['it''s.here']",
		},
	}

	for _, tc := range tests {
		node := dynPathToStructPath(tc.path)
		assert.Equal(t, tc.expected, node.String())
	}
}

// extractReferences gates references on the state type: a reference in an input-only field
// (e.g. a bundle:"readonly" field like volumes' volume_path) must not become a dependency,
// while references in state fields (e.g. comment) are still extracted.
func TestExtractReferences_ExcludesReadonlyFields(t *testing.T) {
	adapters, err := dresources.InitAll(nil)
	require.NoError(t, err)
	// The volume state type is catalog.CreateVolumeRequestContent, which has
	// comment but not volume_path.
	stateType := adapters["volumes"].StateType()

	const yml = `
resources:
  volumes:
    v:
      catalog_name: main
      schema_name: myschema
      name: myvol
      comment: "${resources.schemas.kept.name}"
      volume_path: "/Volumes/main/${resources.schemas.dropped.name}/myvol"
`
	root, err := yamlloader.LoadYAML("test", bytes.NewBufferString(yml))
	require.NoError(t, err)

	refs, err := extractReferences(root, "resources.volumes.v", stateType)
	require.NoError(t, err)

	assert.Equal(t, map[string]string{
		"comment": "${resources.schemas.kept.name}",
	}, refs)
}

func TestExtractReferences_DoesNotTreatConfigSliceAsStateStruct(t *testing.T) {
	type triggersState struct {
		OnValueChange map[string]string `json:"on_value_change,omitempty"`
	}
	type lifecycleState struct {
		Triggers triggersState `json:"triggers"`
	}
	type state struct {
		Lifecycle lifecycleState `json:"lifecycle"`
	}

	const yml = `
resources:
  job_runs:
    run:
      lifecycle:
        triggers:
          - on_value_change: "${resources.jobs.watched.id}"
`
	root, err := yamlloader.LoadYAML("test", bytes.NewBufferString(yml))
	require.NoError(t, err)

	refs, err := extractReferences(root, "resources.job_runs.run", reflect.TypeFor[*state]())
	require.NoError(t, err)
	assert.Empty(t, refs)
}

func TestShouldSkipBackendDefault_ManagedPropertiesOnly(t *testing.T) {
	// Rules mirror the schemas backend_defaults in resources.yml, but the test is
	// deliberately self-contained so that edits to resources.yml don't break it.
	// The real wiring is covered by acceptance/bundle/resources/schemas/drift.
	managedDefaults, err := structpath.ParsePattern("properties['unity.catalog.managed.*.defaults.*']")
	require.NoError(t, err)
	cfg := &dresources.ResourceLifecycleConfig{
		BackendDefaults: []dresources.BackendDefaultRule{
			{Field: managedDefaults},
		},
	}

	tests := []struct {
		name     string
		path     string
		remote   any
		expected bool
	}{
		{
			name:     "managed delta row tracking property",
			path:     "properties['unity.catalog.managed.delta.defaults.delta.enableRowTracking']",
			remote:   "true",
			expected: true,
		},
		{
			name:     "managed iceberg catalog property",
			path:     "properties['unity.catalog.managed.iceberg.defaults.delta.feature.catalogManaged']",
			remote:   "true",
			expected: true,
		},
		{
			name:     "managed delta cluster by auto property",
			path:     "properties['unity.catalog.managed.delta.defaults.defaultClusterByAuto']",
			remote:   "true",
			expected: true,
		},
		{
			name:     "managed delta checkpoint policy property",
			path:     "properties['unity.catalog.managed.delta.defaults.delta.checkpointPolicy']",
			remote:   "v2",
			expected: true,
		},
		{
			name:     "managed delta feature catalogManaged property",
			path:     "properties['unity.catalog.managed.delta.defaults.delta.feature.catalogManaged']",
			remote:   "supported",
			expected: true,
		},
		{
			name:     "unmanaged remote-only property is not skipped",
			path:     "properties['custom.remote_only']",
			remote:   "true",
			expected: false,
		},
		{
			name:     "managed prefix without defaults segment is not skipped",
			path:     "properties['unity.catalog.managed.delta.other.delta.enableRowTracking']",
			remote:   "true",
			expected: false,
		},
		{
			name:     "managed-only parent properties map is skipped",
			path:     "properties",
			remote:   map[string]string{"unity.catalog.managed.delta.defaults.delta.enableRowTracking": "true"},
			expected: true,
		},
		{
			name:     "mixed parent properties map is not skipped",
			path:     "properties",
			remote:   map[string]string{"unity.catalog.managed.delta.defaults.delta.enableRowTracking": "true", "custom.remote_only": "true"},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path, err := structpath.ParsePath(tt.path)
			require.NoError(t, err)

			reason, ok := shouldSkipBackendDefault(cfg, path, &deployplan.ChangeDesc{
				Old:    nil,
				New:    nil,
				Remote: tt.remote,
			})

			assert.Equal(t, tt.expected, ok)
			if tt.expected {
				assert.Equal(t, deployplan.ReasonBackendDefault, reason)
			} else {
				assert.Empty(t, reason)
			}
		})
	}
}

// A field present in StateType but absent from RemoteType (input-only, e.g. external
// locations' skip_validation) reads back nil, so a coincidental new == remote must NOT
// trip RemoteAlreadySet — that would skip a real local change. A field present in RemoteType
// (jobs' max_concurrent_runs) is not affected.
func TestIsFieldMissingInRemote(t *testing.T) {
	adapters, err := dresources.InitAll(nil)
	require.NoError(t, err)

	tests := []struct {
		resource string
		path     string
		expected bool
	}{
		{"external_locations", "skip_validation", true},
		{"jobs", "max_concurrent_runs", false},
		{"jobs", "name", false},
	}

	for _, tt := range tests {
		t.Run(tt.resource+"."+tt.path, func(t *testing.T) {
			adapter, ok := adapters[tt.resource]
			require.True(t, ok)
			path, err := structpath.ParsePath(tt.path)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, isFieldMissingInRemote(adapter, path))
		})
	}
}

// TestRemoteAlreadySetGuards drives addPerFieldActions directly to pin the two guards that
// keep the RemoteAlreadySet shortcut from swallowing a real local change when the remote
// value is fabricated (read back as a fixed nil/zero rather than the true remote value):
//   - a field declared ignore_remote_changes (present in RemoteType, e.g. pipelines' run_as)
//   - a field absent from RemoteType (missing_in_remote, e.g. external_locations' skip_validation)
//
// In both cases a genuine local change (old != new) that coincidentally makes new == remote
// must still classify as Update, not Skip/remote_already_set.
func TestRemoteAlreadySetGuards(t *testing.T) {
	adapters, err := dresources.InitAll(nil)
	require.NoError(t, err)

	runAs := &pipelines.RunAs{UserName: "someone@example.test"}

	tests := []struct {
		name           string
		resource       string
		field          string
		ch             *deployplan.ChangeDesc
		expectedAction deployplan.ActionType
		expectedReason string
	}{
		{
			// ignore_remote_changes guard: run_as read back nil, config changed A -> B.
			name:           "ignore_remote_changes real local change",
			resource:       "pipelines",
			field:          "run_as",
			ch:             &deployplan.ChangeDesc{Old: runAs, New: nil, Remote: nil},
			expectedAction: deployplan.Update,
		},
		{
			// missing_in_remote guard: skip_validation absent from RemoteType (remote nil),
			// config unset the field it previously had.
			name:           "missing_in_remote real local change",
			resource:       "external_locations",
			field:          "skip_validation",
			ch:             &deployplan.ChangeDesc{Old: true, New: nil, Remote: nil},
			expectedAction: deployplan.Update,
		},
		{
			// Control: a real remote value that matches config is correctly skipped.
			name:           "genuine remote_already_set is skipped",
			resource:       "jobs",
			field:          "max_concurrent_runs",
			ch:             &deployplan.ChangeDesc{Old: 1, New: 2, Remote: 2},
			expectedAction: deployplan.Skip,
			expectedReason: deployplan.ReasonRemoteAlreadySet,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			adapter, ok := adapters[tt.resource]
			require.True(t, ok)
			changes := deployplan.Changes{tt.field: tt.ch}
			err := addPerFieldActions(t.Context(), adapter, changes, nil)
			require.NoError(t, err)
			assert.Equal(t, tt.expectedAction, tt.ch.Action)
			if tt.expectedReason != "" {
				assert.Equal(t, tt.expectedReason, tt.ch.Reason)
			} else {
				assert.NotEqual(t, deployplan.ReasonRemoteAlreadySet, tt.ch.Reason, "a real local change must not be skipped as remote_already_set")
			}
		})
	}
}

func TestShouldSkipWhenRemoved(t *testing.T) {
	cfg := dresources.GetResourceConfig("job_runs")
	for _, tt := range []struct {
		name     string
		path     string
		newValue any
		expected bool
	}{
		{"on_bundle_deploy removed", "lifecycle.triggers.on_bundle_deploy", "", true},
		{"on_bundle_deploy changed", "lifecycle.triggers.on_bundle_deploy", "new-uuid", false},
		{"on_file_change removed", "lifecycle.triggers.on_file_change", nil, true},
		{
			"on_file_change map entry removed",
			"lifecycle.triggers.on_file_change",
			map[string]string{"a.txt": "hash"},
			false,
		},
		{"on_file_change entry removed", "lifecycle.triggers.on_file_change['a.txt']", nil, false},
		{"on_value_change removed", "lifecycle.triggers.on_value_change", nil, true},
		{
			"on_value_change map entry removed",
			"lifecycle.triggers.on_value_change",
			map[string]string{"${var.a}": "a"},
			true,
		},
		{
			"on_value_change remaining entry changed",
			"lifecycle.triggers.on_value_change",
			map[string]string{"${var.a}": "b"},
			false,
		},
		{"on_value_change entry removed", "lifecycle.triggers.on_value_change['${var.a}']", nil, true},
		{"on_value_change entry changed", "lifecycle.triggers.on_value_change['${var.a}']", "new", false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			path := structpath.MustParsePath(tt.path)
			oldValue := any("old")
			if path.String() == "lifecycle.triggers.on_file_change" && tt.newValue != nil {
				oldValue = map[string]string{"a.txt": "hash", "b.txt": "hash"}
			}
			if path.String() == "lifecycle.triggers.on_value_change" && tt.newValue != nil {
				oldValue = map[string]string{"${var.a}": "a", "${var.b}": "b"}
			}
			reason, ok := shouldSkipWhenRemoved(cfg, path, &deployplan.ChangeDesc{Old: oldValue, New: tt.newValue})
			assert.Equal(t, tt.expected, ok)
			if tt.expected {
				assert.Equal(t, "trigger removed", reason)
			}
		})
	}
}

// Map drift handling synthesizes child paths to match against rules. structdiff
// always emits map keys in bracket notation, so synthetic child paths must too;
// otherwise rules wouldn't match for identifier-like keys.
func TestShouldSkipBackendDefault_MapDriftUsesBracketKeys(t *testing.T) {
	field, err := structpath.ParsePattern("properties['simple']")
	require.NoError(t, err)
	cfg := &dresources.ResourceLifecycleConfig{
		BackendDefaults: []dresources.BackendDefaultRule{{Field: field}},
	}

	path, err := structpath.ParsePath("properties")
	require.NoError(t, err)

	reason, ok := shouldSkipBackendDefault(cfg, path, &deployplan.ChangeDesc{
		Remote: map[string]string{"simple": "v"},
	})
	assert.True(t, ok)
	assert.Equal(t, deployplan.ReasonBackendDefault, reason)
}

const jobRunKey = "resources.job_runs.my_run"

// The plan skips only a run that reached the required SUCCESS, so a reference
// served from the remote state cache reads a finished run.
func TestLookupReferencePreDeploy_FinishedJobRun(t *testing.T) {
	b := bundleWithSkippedJobRun(t, &dresources.JobRunRemote{
		RunId:       123,
		ResultState: jobs.RunResultStateSuccess,
		State: &jobs.RunState{
			LifeCycleState: jobs.RunLifeCycleStateTerminated,
			ResultState:    jobs.RunResultStateSuccess,
		},
	})

	value, err := b.LookupReferencePreDeploy(t.Context(), structpath.MustParsePath(jobRunKey+".state.result_state"))

	require.NoError(t, err)
	assert.Equal(t, jobs.RunResultStateSuccess, value)
}

// jobRunResultStateAction classifies the result_state drift of a run in the given
// state. Old == New because a deploy records the outcome it asked for, and only
// the remote says whether the run reached it.
func jobRunResultStateAction(t *testing.T, state *jobs.RunState) *deployplan.ChangeDesc {
	t.Helper()
	adapters, err := dresources.InitAll(nil)
	require.NoError(t, err)

	remote := &dresources.JobRunRemote{RunId: 123, ResultState: state.ResultState, State: state}
	changes := deployplan.Changes{"result_state": &deployplan.ChangeDesc{
		Old:    jobs.RunResultStateSuccess,
		New:    jobs.RunResultStateSuccess,
		Remote: state.ResultState,
	}}

	require.NoError(t, addPerFieldActions(t.Context(), adapters["job_runs"], changes, remote))
	return changes["result_state"]
}

// References are served from the remote state cache only for a run the plan
// skips, so a run that stopped without succeeding has to be re-triggered.
func TestJobRunFinishedWithoutSuccessIsRecreate(t *testing.T) {
	for name, state := range map[string]*jobs.RunState{
		"FAILED": {
			LifeCycleState: jobs.RunLifeCycleStateTerminated,
			ResultState:    jobs.RunResultStateFailed,
		},
		"CANCELED": {
			LifeCycleState: jobs.RunLifeCycleStateTerminated,
			ResultState:    jobs.RunResultStateCanceled,
		},
		"TIMEDOUT": {
			LifeCycleState: jobs.RunLifeCycleStateTerminated,
			ResultState:    jobs.RunResultStateTimedout,
		},
		// A skipped run never ran, so it reports no result_state at all — the same
		// as a run still going. Only the lifecycle state tells the two apart.
		"SKIPPED": {LifeCycleState: jobs.RunLifeCycleStateSkipped},
	} {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, deployplan.Recreate, jobRunResultStateAction(t, state).Action)
		})
	}
}

// A run that has not stopped yet may still succeed, so the plan leaves it
// alone rather than recreating it. Skip does not resume an abandoned wait.
func TestJobRunInProgressIsSkip(t *testing.T) {
	for _, lifeCycleState := range []jobs.RunLifeCycleState{
		jobs.RunLifeCycleStatePending,
		jobs.RunLifeCycleStateRunning,
		jobs.RunLifeCycleStateTerminating,
	} {
		t.Run(string(lifeCycleState), func(t *testing.T) {
			change := jobRunResultStateAction(t, &jobs.RunState{LifeCycleState: lifeCycleState})

			assert.Equal(t, deployplan.Skip, change.Action)
			assert.Equal(t, "run in progress", change.Reason)
		})
	}
}

// The run reached the required outcome, so the plan can skip it.
func TestJobRunSucceededOutcomeIsNotDrift(t *testing.T) {
	change := jobRunResultStateAction(t, &jobs.RunState{
		LifeCycleState: jobs.RunLifeCycleStateTerminated,
		ResultState:    jobs.RunResultStateSuccess,
	})

	assert.Equal(t, deployplan.Skip, change.Action)
}

// bundleWithSkippedJobRun sets up a deploy whose plan skips the run, so
// references to it resolve from the remote state cache. The state cache holds
// PrepareState's output, as a real plan does.
func bundleWithSkippedJobRun(t *testing.T, remote *dresources.JobRunRemote) *DeploymentBundle {
	t.Helper()

	adapters, err := dresources.InitAll(nil)
	require.NoError(t, err)

	plan := deployplan.NewPlanDirect()
	plan.Plan[jobRunKey] = &deployplan.PlanEntry{
		Action:   deployplan.Skip,
		NewState: &structvar.StructVarJSON{},
	}

	b := &DeploymentBundle{Adapters: adapters, Plan: plan}
	state := (&dresources.ResourceJobRun{}).PrepareState(&resources.JobRun{})
	b.StateCache.Store(jobRunKey, structvar.NewStructVar(state, nil))
	b.RemoteStateCache.Store(jobRunKey, remote)
	return b
}

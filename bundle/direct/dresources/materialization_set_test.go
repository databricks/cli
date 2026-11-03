package dresources

import (
	"errors"
	"testing"

	"github.com/databricks/cli/bundle/deployplan"
	"github.com/databricks/cli/libs/structs/structpath"
	"github.com/databricks/databricks-sdk-go/apierr"
	"github.com/databricks/databricks-sdk-go/service/ml"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// operationStub stands in for a generated operation interface. The three verbs' interfaces differ
// only in what Wait returns, which materializationSetOperationTarget never calls.
type operationStub struct {
	name     string
	metadata *ml.MaterializationSetOperationMetadata
	err      error
}

func (s operationStub) Name() string { return s.name }

func (s operationStub) Metadata() (*ml.MaterializationSetOperationMetadata, error) {
	return s.metadata, s.err
}

// The set id names what a failed create, update or delete was changing. The server assigns it
// before reserving the operation record, so the fallback to the operation name covers the cases
// where the metadata is unreadable rather than a set that has no id.
func TestMaterializationSetOperationTarget(t *testing.T) {
	assert.Equal(t, "materialization set set-1", materializationSetOperationTarget(operationStub{
		name:     "operations/abc",
		metadata: &ml.MaterializationSetOperationMetadata{MaterializationSetId: "set-1"},
	}))

	assert.Equal(t, "operation operations/abc", materializationSetOperationTarget(operationStub{
		name:     "operations/abc",
		metadata: &ml.MaterializationSetOperationMetadata{},
	}))

	assert.Equal(t, "operation operations/abc", materializationSetOperationTarget(operationStub{
		name: "operations/abc",
	}))

	assert.Equal(t, "operation operations/abc", materializationSetOperationTarget(operationStub{
		name: "operations/abc",
		err:  errors.New("metadata is not a MaterializationSetOperationMetadata"),
	}))
}

func TestMaterializationSetChangeFailuresFromUpdateProgress(t *testing.T) {
	failures := materializationSetChangeFailures(&ml.MaterializationSetOperationMetadata{
		UpdateProgress: &ml.MaterializationSetUpdateProgress{
			DeletedMaterializedFeatureIds: []string{"mf-gone"},
			FailedMaterializedFeatures: []ml.MaterializedFeatureChangeFailure{
				{
					MaterializedFeatureId: "mf-kept",
					ErrorCode:             "PERMISSION_DENIED",
					Message:               "no permission on the online table",
				},
			},
		},
	})

	assert.Equal(t, []string{"mf-kept (PERMISSION_DENIED): no permission on the online table"}, failures)
}

func TestMaterializationSetChangeFailuresFromDeletionProgress(t *testing.T) {
	failures := materializationSetChangeFailures(&ml.MaterializationSetOperationMetadata{
		DeletionProgress: &ml.MaterializationSetDeletionProgress{
			FailedMaterializedFeatures: []ml.MaterializedFeatureChangeFailure{
				{MaterializedFeatureId: "mf-1", ErrorCode: "INTERNAL_ERROR", Message: "pipeline stuck"},
				{MaterializedFeatureId: "mf-2", ErrorCode: "RESOURCE_CONFLICT", Message: "in use"},
			},
		},
	})

	assert.Equal(t, []string{
		"mf-1 (INTERNAL_ERROR): pipeline stuck",
		"mf-2 (RESOURCE_CONFLICT): in use",
	}, failures)
}

func TestMaterializationSetChangeFailuresWithoutFailedMembers(t *testing.T) {
	// A creation carries no progress arm at all, and a change that fully applied carries one
	// naming no failure; neither has a member to report.
	assert.Empty(t, materializationSetChangeFailures(&ml.MaterializationSetOperationMetadata{}))
	assert.Empty(t, materializationSetChangeFailures(&ml.MaterializationSetOperationMetadata{
		DeletionProgress: &ml.MaterializationSetDeletionProgress{
			DeletedMaterializedFeatureIds: []string{"mf-1"},
		},
	}))
	assert.Empty(t, materializationSetChangeFailures(nil))
}

func TestMaterializationSetOperationFailureNamesMembersLeftUnchanged(t *testing.T) {
	waitErr := errors.New("operation failed: [INTERNAL_ERROR] Changed 0 of 1 materialized features")
	err := materializationSetOperationFailure(operationStub{
		name: "operations/abc",
		metadata: &ml.MaterializationSetOperationMetadata{
			MaterializationSetId: "set-1",
			UpdateProgress: &ml.MaterializationSetUpdateProgress{
				FailedMaterializedFeatures: []ml.MaterializedFeatureChangeFailure{
					{MaterializedFeatureId: "mf-1", ErrorCode: "INTERNAL_ERROR", Message: "pipeline stuck"},
				},
			},
		},
	}, waitErr)

	assert.EqualError(t, err, "materialization set set-1: operation failed: [INTERNAL_ERROR] Changed 0 of 1 "+
		"materialized features; members left unchanged: mf-1 (INTERNAL_ERROR): pipeline stuck")
	assert.ErrorIs(t, err, waitErr)
	_, isAPIError := errors.AsType[*apierr.APIError](err)
	assert.False(t, isAPIError)
}

func TestMaterializationSetOperationFailureWithoutFailedMembers(t *testing.T) {
	waitErr := errors.New("operation failed: [INVALID_PARAMETER_VALUE] Batch and streaming features cannot be mixed")
	err := materializationSetOperationFailure(operationStub{
		name:     "operations/abc",
		metadata: &ml.MaterializationSetOperationMetadata{MaterializationSetId: "set-1"},
	}, waitErr)

	assert.EqualError(t, err, "materialization set set-1: operation failed: [INVALID_PARAMETER_VALUE] Batch and "+
		"streaming features cannot be mixed")
	assert.ErrorIs(t, err, waitErr)
}

// TestMaterializationSetMembershipChangeAction enumerates every path shape the planner hands this
// resource: a whole member and the whole member list are an update unless they hold a member the
// deployed set lacks, and every other path keeps the action the rules gave it.
func TestMaterializationSetMembershipChangeAction(t *testing.T) {
	const member = "materialized_features[member='feature_name=main.ns.feat|online_store_name=store|is_streaming_mode=false']"
	const streamingMember = "materialized_features[member='feature_name=main.ns.feat|online_store_name=store|is_streaming_mode=true']"
	const movedStoreMember = "materialized_features[member='feature_name=main.ns.feat|online_store_name=other|is_streaming_mode=false']"
	const addedMember = "materialized_features[member='feature_name=main.ns.other|is_streaming_mode=false']"

	deployed := ml.MaterializedFeature{
		FeatureName:       "main.ns.feat",
		OnlineStoreConfig: &ml.OnlineStoreConfig{OnlineStoreName: "store"},
	}
	added := ml.MaterializedFeature{FeatureName: "main.ns.other"}
	turnedStreaming := deployed
	turnedStreaming.StreamingMode = &ml.StreamingMode{}
	movedStore := deployed
	movedStore.OnlineStoreConfig = &ml.OnlineStoreConfig{OnlineStoreName: "other"}
	remote := &ml.MaterializationSet{MaterializedFeatures: []ml.MaterializedFeature{deployed}}

	for _, test := range []struct {
		name   string
		path   string
		new    any
		remote *ml.MaterializationSet
		action deployplan.ActionType
		reason string
	}{
		{"member the deployed set lacks", addedMember, added, remote, deployplan.Recreate, reasonMemberAdded},
		{"member whose trigger turned streaming", streamingMember, turnedStreaming, remote, deployplan.Recreate, reasonMemberAdded},
		{"member whose online store changed", movedStoreMember, movedStore, remote, deployplan.Recreate, reasonMemberAdded},
		{"member the deployed set has", member, deployed, remote, deployplan.Update, "immutable"},
		{"member removed", member, nil, remote, deployplan.Update, "immutable"},
		{"list holding a member the deployed set lacks", "materialized_features", []ml.MaterializedFeature{deployed, added}, remote, deployplan.Recreate, reasonMemberAdded},
		{"list the deployed set has", "materialized_features", []ml.MaterializedFeature{deployed}, remote, deployplan.Update, "immutable"},
		{"list emptied", "materialized_features", nil, remote, deployplan.Update, "immutable"},
		{"set missing remotely", addedMember, added, nil, deployplan.Update, "immutable"},
		{"feature name field", member + ".feature_name", nil, remote, deployplan.Recreate, "immutable"},
		{"trigger field", member + ".cron_schedule_trigger", nil, remote, deployplan.Recreate, "immutable"},
		{"destination field", member + ".online_store_config.table_name_prefix", nil, remote, deployplan.Recreate, "immutable"},
		{"set id", "materialization_set_id", nil, remote, deployplan.Recreate, "immutable"},
	} {
		t.Run(test.name, func(t *testing.T) {
			path, err := structpath.ParsePath(test.path)
			require.NoError(t, err)

			change := &ChangeDesc{Action: deployplan.Recreate, Reason: "immutable", New: test.new}
			require.NoError(t, (&ResourceMaterializationSet{}).OverrideChangeDesc(
				t.Context(), path, change, test.remote))
			assert.Equal(t, test.action, change.Action)
			assert.Equal(t, test.reason, change.Reason)
		})
	}
}

// TestMaterializationSetDestinationSuppressionStillReachesMemberFields keeps the two blocks of
// OverrideChangeDesc disjoint: widening the membership carve-out to one segment must not shadow the
// input-only suppression, which only ever sees a member's destination three segments down.
func TestMaterializationSetDestinationSuppressionStillReachesMemberFields(t *testing.T) {
	path, err := structpath.ParsePath(
		"materialized_features[member='feature_name=main.ns.feat|is_streaming_mode=false'].offline_store_config")
	require.NoError(t, err)

	change := &ChangeDesc{Action: deployplan.Recreate, Reason: "spec:immutable", New: "declared"}
	require.NoError(t, (&ResourceMaterializationSet{}).OverrideChangeDesc(
		t.Context(), path, change, nil))

	assert.Equal(t, deployplan.Skip, change.Action)
	assert.Equal(t, "input_only_not_returned", change.Reason)
}

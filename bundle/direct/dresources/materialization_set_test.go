package dresources

import (
	"errors"
	"testing"

	"github.com/databricks/cli/bundle/deployplan"
	"github.com/databricks/cli/libs/structs/structpath"
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

// TestMaterializationSetMembershipChangeIsAnUpdate enumerates every path shape the planner hands
// this resource, so the membership carve-out's reach is visible rather than inferred. The two
// shapes it covers are a member added or removed whole and the member list emptied whole; every
// other path keeps the action the rules gave it.
func TestMaterializationSetMembershipChangeIsAnUpdate(t *testing.T) {
	const member = "materialized_features[member='feature_name=main.ns.feat|online_store_name=store|is_streaming_mode=false']"

	for _, test := range []struct {
		path   string
		action deployplan.ActionType
	}{
		{"materialized_features", deployplan.Update},
		{member, deployplan.Update},
		{"materialized_features[member='feature_name=main.ns.feat|is_streaming_mode=false']", deployplan.Update},
		{member + ".feature_name", deployplan.Recreate},
		{member + ".cron_schedule_trigger", deployplan.Recreate},
		{member + ".online_store_config.table_name_prefix", deployplan.Recreate},
		{"materialization_set_id", deployplan.Recreate},
	} {
		t.Run(test.path, func(t *testing.T) {
			path, err := structpath.ParsePath(test.path)
			require.NoError(t, err)

			change := &ChangeDesc{Action: deployplan.Recreate, Reason: "immutable"}
			require.NoError(t, (&ResourceMaterializationSet{}).OverrideChangeDesc(
				t.Context(), path, change, nil))
			assert.Equal(t, test.action, change.Action)
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

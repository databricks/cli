package dresources

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"uuid"

	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/bundle/deployplan"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/cli/libs/structs/structpath"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/common/types/fieldmask"
	"github.com/databricks/databricks-sdk-go/service/ml"
)

// https://docs.databricks.com/api/workspace/featureengineering
type ResourceMaterializationSet struct {
	client *databricks.WorkspaceClient
}

func (*ResourceMaterializationSet) New(client *databricks.WorkspaceClient) *ResourceMaterializationSet {
	return &ResourceMaterializationSet{client: client}
}

func (*ResourceMaterializationSet) PrepareState(input *resources.MaterializationSet) *ml.MaterializationSet {
	return &input.MaterializationSet
}

func (*ResourceMaterializationSet) KeyedSlices() map[string]any {
	return map[string]any{
		"materialized_features": resources.MaterializedFeatureMemberKey,
	}
}

var (
	materializedFeaturesPattern = structpath.NewPatternDotString(nil, "materialized_features")
	materializedFeatureMember   = structpath.NewPatternBracketStar(materializedFeaturesPattern)

	materializedFeatureMemberFeatureName = structpath.NewPatternDotString(materializedFeatureMember, "feature_name")

	// The destination fields the server accepts on write and never returns on read. The whole
	// offline block is input-only; online_store_name is returned, so it is deliberately absent.
	redactedDestinationPatterns = []*structpath.PatternNode{
		structpath.NewPatternDotString(materializedFeatureMember, "offline_store_config"),
		structpath.NewPatternDotString(
			structpath.NewPatternDotString(materializedFeatureMember, "online_store_config"),
			"catalog_name",
		),
		structpath.NewPatternDotString(
			structpath.NewPatternDotString(materializedFeatureMember, "online_store_config"),
			"schema_name",
		),
		structpath.NewPatternDotString(
			structpath.NewPatternDotString(materializedFeatureMember, "online_store_config"),
			"table_name_prefix",
		),
	}
)

// isRedactedDestinationField reports whether path names a destination field the read never returns.
func isRedactedDestinationField(path *structpath.PathNode) bool {
	return slices.ContainsFunc(redactedDestinationPatterns, path.HasPatternPrefix)
}

// isAbsentRemotely reports whether a read returned nothing for a field, which for a redacted field
// is indistinguishable from the server having no value.
func isAbsentRemotely(remote any) bool {
	if remote == nil {
		return true
	}
	text, isString := remote.(string)
	return isString && text == ""
}

// reasonMemberAdded marks a recreate caused by a configured member the deployed set lacks.
const reasonMemberAdded = "member_added"

// reasonFeatureNameCaseOnly marks a skip of a member's feature name that differs from the deployed
// one only in case.
const reasonFeatureNameCaseOnly = "feature_name_case_only"

// OverrideChangeDesc plans a member removed from the set as an update and a member the deployed
// set lacks as a recreate, skips a member's feature name that differs only in case, and suppresses
// the drift left behind by an ignore_remote_changes rule that cannot fire.
func (*ResourceMaterializationSet) OverrideChangeDesc(ctx context.Context, path *structpath.PathNode, change *ChangeDesc, remote *ml.MaterializationSet) error {
	// The immutable field rules also match a whole member, materialized_features[member=...], and
	// the whole list, materialized_features, because the matcher truncates a longer pattern. The
	// update endpoint removes members but cannot add one, so only a configured member the deployed
	// set lacks recreates the set.
	if path.Len() <= 2 && path.HasPatternPrefix(materializedFeaturesPattern) {
		added := membersAbsentRemotely(change.New, remote)
		if len(added) == 0 {
			change.Action = deployplan.Update
			return nil
		}
		for _, member := range added {
			log.Warnf(ctx, "the member for feature %q is not in the deployed set, and members cannot be "+
				"added to an existing set, so the whole set will be recreated: every member is dropped "+
				"and recomputed", member.FeatureName)
		}
		change.Action = deployplan.Recreate
		change.Reason = reasonMemberAdded
		return nil
	}

	// A paired member's feature names differ only in case, because MaterializedFeatureMemberKey
	// lowercases them.
	if path.HasPatternPrefix(materializedFeatureMemberFeatureName) {
		change.Action = deployplan.Skip
		change.Reason = reasonFeatureNameCaseOnly
		return nil
	}

	// The ignore_remote_changes suppression on a destination field cannot fire for this resource,
	// which keys its member slice on a composite key defined by materializationSetMemberKey. The
	// suppression fires only when the persisted value equals the configured
	// one, and the persisted-value lookup resolves that composite against the element's field names,
	// which never match, so it always returns nothing. An unset change.Old is the surrogate for the
	// persisted value being unreachable; a repoint the config asked for carries that value in from
	// the local diff instead, and so still recreates.
	if isRedactedDestinationField(path) && change.Old == nil && change.New != nil &&
		isAbsentRemotely(change.Remote) {
		change.Action = deployplan.Skip
		change.Reason = "input_only_not_returned"
	}
	return nil
}

// membersAbsentRemotely returns the configured members, given one member or the whole list, whose
// key no member of the remote set has. A set missing remotely has none, because the planner then
// creates the set instead of recreating it.
func membersAbsentRemotely(configured any, remote *ml.MaterializationSet) []ml.MaterializedFeature {
	if remote == nil {
		return nil
	}
	var members []ml.MaterializedFeature
	switch value := configured.(type) {
	case ml.MaterializedFeature:
		members = []ml.MaterializedFeature{value}
	case []ml.MaterializedFeature:
		members = value
	}

	remoteKeys := make(map[string]bool, len(remote.MaterializedFeatures))
	for _, member := range remote.MaterializedFeatures {
		_, key := resources.MaterializedFeatureMemberKey(member)
		remoteKeys[key] = true
	}
	var absent []ml.MaterializedFeature
	for _, member := range members {
		if _, key := resources.MaterializedFeatureMemberKey(member); !remoteKeys[key] {
			absent = append(absent, member)
		}
	}
	return absent
}

func (r *ResourceMaterializationSet) DoRead(ctx context.Context, id string) (*ml.MaterializationSet, error) {
	return r.client.FeatureEngineering.GetMaterializationSet(ctx, ml.GetMaterializationSetRequest{
		MaterializationSetId: id,
	})
}

// withLowercaseFeatureNames returns a copy of the set with each member's feature name lowercased,
// because the service rejects a member not named by its feature's resolved, lowercase name.
func withLowercaseFeatureNames(set ml.MaterializationSet) ml.MaterializationSet {
	set.MaterializedFeatures = slices.Clone(set.MaterializedFeatures)
	for i := range set.MaterializedFeatures {
		set.MaterializedFeatures[i].FeatureName = strings.ToLower(set.MaterializedFeatures[i].FeatureName)
	}
	return set
}

// DoCreate submits the set and waits for the long-running operation the create returns. The created
// set arrives as that operation's response, which Wait reaches by polling GetOperation.
func (r *ResourceMaterializationSet) DoCreate(ctx context.Context, config *ml.MaterializationSet) (string, *ml.MaterializationSet, error) {
	// The endpoint takes an idempotency token, and a repeated request carrying one already used
	// returns the operation the first request with the same idempotency token created. Minted here
	// so the token the server sees is the one this call chose.
	operation, err := r.client.FeatureEngineering.CreateMaterializationSet(ctx, ml.CreateMaterializationSetRequest{
		MaterializationSet: withLowercaseFeatureNames(*config),
		RequestId:          uuid.New().String(),
		ForceSendFields:    nil,
	})
	if err != nil {
		return "", nil, err
	}

	set, err := operation.Wait(ctx)
	if err != nil {
		return "", nil, materializationSetOperationFailure(operation, err)
	}
	// The id is what the engine records the resource under, so an operation that settles without
	// one is a failure even though the wait succeeded.
	if set == nil || set.MaterializationSetId == "" {
		return "", nil, fmt.Errorf("%s: creation finished without returning the set",
			materializationSetOperationTarget(operation))
	}
	return set.MaterializationSetId, set, nil
}

// materializationSetChangeFailures names each materialized feature member a partly applied change
// in the update or delete action could not reach, from whichever progress arm the operation's
// metadata carries. Empty for a creation action, and for a failure (e.g. a validation error) raised
// before the server performed the action.
func materializationSetChangeFailures(metadata *ml.MaterializationSetOperationMetadata) []string {
	if metadata == nil {
		return nil
	}
	var members []ml.MaterializedFeatureChangeFailure
	switch {
	case metadata.UpdateProgress != nil:
		members = metadata.UpdateProgress.FailedMaterializedFeatures
	case metadata.DeletionProgress != nil:
		members = metadata.DeletionProgress.FailedMaterializedFeatures
	}

	failures := make([]string, 0, len(members))
	for _, member := range members {
		failures = append(failures, fmt.Sprintf("%s (%s): %s",
			member.MaterializedFeatureId, member.ErrorCode, member.Message))
	}
	return failures
}

// materializationSetOperationFailure names the set a failed wait was changing and appends each
// member the change left unchanged. The wait error already carries the operation's error code and
// message.
func materializationSetOperationFailure(operation materializationSetOperation, waitErr error) error {
	err := fmt.Errorf("%s: %w", materializationSetOperationTarget(operation), waitErr)
	metadata, _ := operation.Metadata()
	if failures := materializationSetChangeFailures(metadata); len(failures) > 0 {
		return fmt.Errorf("%w; members left unchanged: %s", err, strings.Join(failures, "; "))
	}
	return err
}

// materializationSetOperation is what a create, update and delete operation have in common: the
// three generated interfaces share these two methods and differ only in what Wait returns.
type materializationSetOperation interface {
	Name() string
	Metadata() (*ml.MaterializationSetOperationMetadata, error)
}

// materializationSetOperationTarget names the set an operation was changing, taking the set id from
// the operation metadata when the server supplied it and falling back to the operation name.
func materializationSetOperationTarget(operation materializationSetOperation) string {
	metadata, err := operation.Metadata()
	if err == nil && metadata != nil && metadata.MaterializationSetId != "" {
		return "materialization set " + metadata.MaterializationSetId
	}
	return "operation " + operation.Name()
}

// DoUpdate reconciles the set's membership and waits for the long-running operation the update
// returns. The reconciled set arrives as that operation's response.
func (r *ResourceMaterializationSet) DoUpdate(ctx context.Context, id string, config *ml.MaterializationSet, _ *PlanEntry) (*ml.MaterializationSet, error) {
	update := withLowercaseFeatureNames(*config)
	update.MaterializationSetId = id

	operation, err := r.client.FeatureEngineering.UpdateMaterializationSet(ctx, ml.UpdateMaterializationSetRequest{
		MaterializationSetId: id,
		MaterializationSet:   update,
		UpdateMask:           *fieldmask.New([]string{"*"}),
		RequestId:            uuid.New().String(),
		ForceSendFields:      nil,
	})
	if err != nil {
		return nil, err
	}

	set, err := operation.Wait(ctx)
	if err != nil {
		return nil, materializationSetOperationFailure(operation, err)
	}
	return set, nil
}

// DoDelete deletes the set and waits for the long-running operation the delete returns, which
// settles with no response body.
func (r *ResourceMaterializationSet) DoDelete(ctx context.Context, id string, _ *ml.MaterializationSet) error {
	operation, err := r.client.FeatureEngineering.DeleteMaterializationSet(ctx, ml.DeleteMaterializationSetRequest{
		MaterializationSetId: id,
		RequestId:            uuid.New().String(),
		AllowMissing:         false,
		ForceSendFields:      nil,
	})
	if err != nil {
		return err
	}
	if err := operation.Wait(ctx); err != nil {
		return materializationSetOperationFailure(operation, err)
	}
	return nil
}

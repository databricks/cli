package dresources

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/bundle/deployplan"
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

// materializationSetUpdatableFields is the only mask the update endpoint accepts, so "*" cannot
// be used. Membership is reconciled whole: the server deletes members the request omits.
var materializationSetUpdatableFields = []string{"materialized_features"}

// escapeMemberKeyPart percent-escapes the separator so the join in materializationSetMemberKey is
// injective. A feature name is validated only for length and non-emptiness, so it may contain the
// separator; two members sharing a key would be diffed as one, and the request would then omit a
// member the server deletes because it cannot see it.
func escapeMemberKeyPart(part string) string {
	part = strings.ReplaceAll(part, "%", "%25")
	return strings.ReplaceAll(part, "|", "%7C")
}

// materializationSetMemberKey identifies a materialized feature member by content rather than
// position. The key components are the feature name, the online store name (empty for the offline
// member), and whether the trigger is streaming.
func materializationSetMemberKey(member ml.MaterializedFeature) (string, string) {
	onlineStoreName := ""
	if member.OnlineStoreConfig != nil {
		onlineStoreName = member.OnlineStoreConfig.OnlineStoreName
	}
	return "member", strings.Join([]string{
		escapeMemberKeyPart(member.FeatureName),
		escapeMemberKeyPart(onlineStoreName),
		strconv.FormatBool(member.StreamingMode != nil),
	}, "|")
}

func (*ResourceMaterializationSet) KeyedSlices() map[string]any {
	return map[string]any{
		"materialized_features": materializationSetMemberKey,
	}
}

var (
	materializedFeaturesPattern = structpath.NewPatternDotString(nil, "materialized_features")
	materializedFeatureMember   = structpath.NewPatternBracketStar(materializedFeaturesPattern)

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
	for _, pattern := range redactedDestinationPatterns {
		if path.HasPatternPrefix(pattern) {
			return true
		}
	}
	return false
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

// OverrideChangeDesc does two things the generic classification gets wrong here: it marks a whole
// materialized feature member added to or removed from the set as an update rather than a
// destroy-and-recreate, and it suppresses the drift left behind by an ignore_remote_changes rule
// that cannot fire.
func (*ResourceMaterializationSet) OverrideChangeDesc(_ context.Context, path *structpath.PathNode, change *ChangeDesc, _ *ml.MaterializationSet) error {
	// A member added to or removed from the set is a two-segment path,
	// materialized_features[member='<feature name>|<online store name>|<is streaming>'], which an
	// immutable field rule under materialized_features[*] still matches. That match counts as an
	// update rather than a destroy-and-recreate, because a whole-member addition or removal is what
	// the update endpoint handles.
	if path.Len() == 2 && path.HasPatternPrefix(materializedFeaturesPattern) {
		change.Action = deployplan.Update
		return nil
	}

	// The ignore_remote_changes suppression on a destination field cannot fire for this resource,
	// which keys its member slice on a composite of feature name, online store name and whether the
	// member is streaming. The suppression fires only when the persisted value equals the configured
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

func (r *ResourceMaterializationSet) DoRead(ctx context.Context, id string) (*ml.MaterializationSet, error) {
	return r.client.FeatureEngineering.GetMaterializationSet(ctx, ml.GetMaterializationSetRequest{
		MaterializationSetId: id,
	})
}

// DoCreate submits the set and waits for the long-running operation the create returns. The created
// set arrives as that operation's response, which Wait reaches by polling GetOperation.
func (r *ResourceMaterializationSet) DoCreate(ctx context.Context, config *ml.MaterializationSet) (string, *ml.MaterializationSet, error) {
	operation, err := r.client.FeatureEngineering.CreateMaterializationSet(ctx, ml.CreateMaterializationSetRequest{
		MaterializationSet: *config,
	})
	if err != nil {
		return "", nil, err
	}

	set, err := operation.Wait(ctx)
	if err != nil {
		return "", nil, fmt.Errorf("%s: %w", materializationSetCreationTarget(operation), err)
	}
	// The id is what the engine records the resource under, so an operation that settles without
	// one is a failure even though the wait succeeded.
	if set == nil || set.MaterializationSetId == "" {
		return "", nil, fmt.Errorf("%s: creation finished without returning the set",
			materializationSetCreationTarget(operation))
	}
	return set.MaterializationSetId, set, nil
}

// materializationSetCreationTarget names what a creation was building, taking the set id from the
// operation metadata when the server supplied it and falling back to the operation name.
func materializationSetCreationTarget(operation ml.CreateMaterializationSetOperationInterface) string {
	metadata, err := operation.Metadata()
	if err == nil && metadata != nil && metadata.MaterializationSetId != "" {
		return "materialization set " + metadata.MaterializationSetId
	}
	return "operation " + operation.Name()
}

func (r *ResourceMaterializationSet) DoUpdate(ctx context.Context, id string, config *ml.MaterializationSet, _ *PlanEntry) (*ml.MaterializationSet, error) {
	update := *config
	update.MaterializationSetId = id

	return r.client.FeatureEngineering.UpdateMaterializationSet(ctx, ml.UpdateMaterializationSetRequest{
		MaterializationSetId: id,
		MaterializationSet:   update,
		UpdateMask:           *fieldmask.New(materializationSetUpdatableFields),
	})
}

func (r *ResourceMaterializationSet) DoDelete(ctx context.Context, id string, _ *ml.MaterializationSet) error {
	return r.client.FeatureEngineering.DeleteMaterializationSet(ctx, ml.DeleteMaterializationSetRequest{
		MaterializationSetId: id,
	})
}

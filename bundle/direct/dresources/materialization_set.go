package dresources

import (
	"context"
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

// materializationSetMemberKey identifies a member by content rather than position, matching how
// UpdateMaterializationSet pairs a requested member with a stored one. The parts are the feature
// name, the online store name (empty for the offline member, since an online destination always
// carries one), and whether the member is streaming. Feature name alone is not unique: one feature
// has an offline member plus one member per online store.
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

var materializedFeaturesPattern = structpath.NewPatternDotString(nil, "materialized_features")

// OverrideChangeDesc keeps a member added or removed whole on the update path. Such a change is a
// two-segment path, which still matches every per-field rule under materialized_features[*] because
// the matcher truncates a pattern longer than the path, so without this it classifies as recreate.
//
// Both directions belong on the update path, for different reasons. A removal is what the endpoint
// is for: it deletes the members the request omits, leaving the rest untouched. An addition the
// endpoint rejects outright, and routing it here surfaces that rejection naming the feature,
// instead of recreating the set and destroying every sibling the config left alone to achieve it.
//
// A one-segment path is deliberately left alone: it means one side has no members at all, which
// materialized_features being required makes invalid config, and recreate is the safe plan for it.
func (*ResourceMaterializationSet) OverrideChangeDesc(_ context.Context, path *structpath.PathNode, change *ChangeDesc, _ *ml.MaterializationSet) error {
	if path.Len() == 2 && path.HasPatternPrefix(materializedFeaturesPattern) {
		change.Action = deployplan.Update
	}
	return nil
}

func (r *ResourceMaterializationSet) DoRead(ctx context.Context, id string) (*ml.MaterializationSet, error) {
	return r.client.FeatureEngineering.GetMaterializationSet(ctx, ml.GetMaterializationSetRequest{
		MaterializationSetId: id,
	})
}

func (r *ResourceMaterializationSet) DoCreate(ctx context.Context, config *ml.MaterializationSet) (string, *ml.MaterializationSet, error) {
	response, err := r.client.FeatureEngineering.CreateMaterializationSet(ctx, ml.CreateMaterializationSetRequest{
		MaterializationSet: *config,
	})
	if err != nil {
		return "", nil, err
	}
	return response.MaterializationSetId, response, nil
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

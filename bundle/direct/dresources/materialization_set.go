package dresources

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/bundle/deployplan"
	"github.com/databricks/cli/libs/structs/structpath"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/apierr"
	"github.com/databricks/databricks-sdk-go/common/types/fieldmask"
	"github.com/databricks/databricks-sdk-go/service/ml"
	"github.com/google/uuid"
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
// position, as labelled components joined by the separator escapeMemberKeyPart escapes.
//
// online_store_name is omitted rather than left empty when the member has no online destination,
// which is what keeps an offline member distinct from an online one whose store name is unset.
//
// The server pairs members by the same components in MaterializationSetMemberKey -- see
// feature-store/src/main/scala/com/databricks/feature-store/utils/materialized-features/MaterializationSetUtils.scala
// -- holding them in a case class rather than a joined string, so this encoding is the CLI's own
// and only the component list has to agree. Adding, dropping or redefining one takes the same
// change there.
func materializationSetMemberKey(member ml.MaterializedFeature) (string, string) {
	parts := []string{"feature_name=" + escapeMemberKeyPart(member.FeatureName)}
	if member.OnlineStoreConfig != nil {
		parts = append(parts,
			"online_store_name="+escapeMemberKeyPart(member.OnlineStoreConfig.OnlineStoreName))
	}
	parts = append(parts, "is_streaming_mode="+strconv.FormatBool(member.StreamingMode != nil))

	return "member", strings.Join(parts, "|")
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
	// A member added or removed whole is a two-segment path, materialized_features[member=...]
	// carrying the key materializationSetMemberKey builds, and the member list emptied whole is the
	// one-segment materialized_features. An immutable field rule
	// under materialized_features[*] matches either, because the matcher truncates a pattern longer
	// than the path. Both count as an update rather than a destroy-and-recreate, because
	// reconciling membership is what the update endpoint does: it refuses an addition by naming the
	// feature that matched no member, and an emptied list by pointing at the delete endpoint.
	if path.Len() <= 2 && path.HasPatternPrefix(materializedFeaturesPattern) {
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
	// Each verb takes an idempotency token, and a repeat carrying one already used returns the
	// operation the first request created rather than acting again. Minted here so the token the
	// server sees is the one this call chose.
	operation, err := r.client.FeatureEngineering.CreateMaterializationSet(ctx, ml.CreateMaterializationSetRequest{
		MaterializationSet: *config,
		RequestId:          uuid.NewString(),
	})
	if err != nil {
		return "", nil, err
	}

	set, err := operation.Wait(ctx)
	if err != nil {
		return "", nil, r.materializationSetOperationFailure(ctx, operation, err)
	}
	// The id is what the engine records the resource under, so an operation that settles without
	// one is a failure even though the wait succeeded.
	if set == nil || set.MaterializationSetId == "" {
		return "", nil, fmt.Errorf("%s: creation finished without returning the set",
			materializationSetOperationTarget(operation))
	}
	return set.MaterializationSetId, set, nil
}

// materializationSetChangeFailures names each member a partly applied change could not reach, from
// whichever progress arm the operation's metadata carries. Empty for a creation, which has no
// per-member outcome, and for a failure raised before the server touched a member.
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

// materializationSetOperationFailure restates a settled-but-failed operation as an API error. The
// generated wait renders the failure's code and message into one string and reads no metadata, so
// the error code is recovered structurally and the per-member progress is appended.
//
// StatusCode is the 200 the call actually returned, which maps to no sentinel: a member that could
// not be torn down must not reach the destroy path's permission-denied leniency, whose purpose is
// to forgive a resource that is already gone.
func (r *ResourceMaterializationSet) materializationSetOperationFailure(
	ctx context.Context,
	operation materializationSetOperation,
	waitErr error,
) error {
	target := materializationSetOperationTarget(operation)
	settled, err := r.client.FeatureEngineering.GetOperation(ctx, ml.GetOperationRequest{
		Name: operation.Name(),
	})
	if err != nil || settled.Error == nil {
		return fmt.Errorf("%s: %w", target, waitErr)
	}

	message := settled.Error.Message
	if metadata, err := operation.Metadata(); err == nil {
		if failures := materializationSetChangeFailures(metadata); len(failures) > 0 {
			message += "; members left unchanged: " + strings.Join(failures, "; ")
		}
	}
	return &apierr.APIError{
		ErrorCode:  string(settled.Error.ErrorCode),
		Message:    target + " operation failed: " + message,
		StatusCode: http.StatusOK,
	}
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
	update := *config
	update.MaterializationSetId = id

	operation, err := r.client.FeatureEngineering.UpdateMaterializationSet(ctx, ml.UpdateMaterializationSetRequest{
		MaterializationSetId: id,
		MaterializationSet:   update,
		UpdateMask:           *fieldmask.New(materializationSetUpdatableFields),
		RequestId:            uuid.NewString(),
	})
	if err != nil {
		return nil, err
	}

	set, err := operation.Wait(ctx)
	if err != nil {
		return nil, r.materializationSetOperationFailure(ctx, operation, err)
	}
	return set, nil
}

// DoDelete deletes the set and waits for the long-running operation the delete returns, which
// settles with no response body.
func (r *ResourceMaterializationSet) DoDelete(ctx context.Context, id string, _ *ml.MaterializationSet) error {
	operation, err := r.client.FeatureEngineering.DeleteMaterializationSet(ctx, ml.DeleteMaterializationSetRequest{
		MaterializationSetId: id,
		RequestId:            uuid.NewString(),
	})
	if err != nil {
		return err
	}
	if err := operation.Wait(ctx); err != nil {
		return r.materializationSetOperationFailure(ctx, operation, err)
	}
	return nil
}

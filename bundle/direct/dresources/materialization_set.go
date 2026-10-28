package dresources

import (
	"context"
	"fmt"
	"net/http"
	"slices"
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

// materializationSetUpdatableFields is the only mask the update endpoint accepts.
var materializationSetUpdatableFields = []string{"materialized_features"}

func (*ResourceMaterializationSet) KeyedSlices() map[string]any {
	return map[string]any{
		"materialized_features": resources.MaterializedFeatureMemberKey,
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
	// reconciling membership is what the update endpoint does.
	if path.Len() <= 2 && path.HasPatternPrefix(materializedFeaturesPattern) {
		change.Action = deployplan.Update
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

func (r *ResourceMaterializationSet) DoRead(ctx context.Context, id string) (*ml.MaterializationSet, error) {
	return r.client.FeatureEngineering.GetMaterializationSet(ctx, ml.GetMaterializationSetRequest{
		MaterializationSetId: id,
	})
}

// DoCreate submits the set and waits for the long-running operation the create returns. The created
// set arrives as that operation's response, which Wait reaches by polling GetOperation.
func (r *ResourceMaterializationSet) DoCreate(ctx context.Context, config *ml.MaterializationSet) (string, *ml.MaterializationSet, error) {
	// The endpoint takes an idempotency token, and a repeated request carrying one already used
	// returns the operation the first request with the same idempotency token created. Minted here
	// so the token the server sees is the one this call chose.
	operation, err := r.client.FeatureEngineering.CreateMaterializationSet(ctx, ml.CreateMaterializationSetRequest{
		MaterializationSet: *config,
		RequestId:          uuid.NewString(),
		ForceSendFields:    nil,
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

// materializationSetOperationFailure restates a response operation with FAILED status as an API
// error.
//
// StatusCode is set to 200 because the create, update and delete endpoints always return an
// operation with FAILED status when an error is raised, except for the errors raised before the
// action is executed, such as a validation error on the request.
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
		ErrorCode:       string(settled.Error.ErrorCode),
		Message:         target + " operation failed: " + message,
		StatusCode:      http.StatusOK,
		ResponseWrapper: nil,
		Details:         nil, //nolint:staticcheck // SA1019: deprecated, but exhaustruct requires every field
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
		ForceSendFields:      nil,
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
		AllowMissing:         false,
		ForceSendFields:      nil,
	})
	if err != nil {
		return err
	}
	if err := operation.Wait(ctx); err != nil {
		return r.materializationSetOperationFailure(ctx, operation, err)
	}
	return nil
}

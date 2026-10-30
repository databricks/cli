package testserver

import (
	"encoding/json"
	"fmt"
	"net/http"
	"slices"
	"strings"

	"github.com/databricks/databricks-sdk-go/service/ml"
)

// This fake mirrors UpdateMaterializationSet closely enough that one set of acceptance goldens
// serves both the local run and a run against a real workspace. Where it rejects, it returns the
// backend's own message text, so a golden recorded locally is a claim about the real API that the
// cloud run then checks.

// materializationSetRefusal is a rule's verdict, separated from how it is reported: a rule the
// backend reaches before reserving an operation record becomes an HTTP error, and one it reaches
// after becomes a settled FAILED operation.
type materializationSetRefusal struct {
	errorCode string
	message   string
}

func materializationSetError(statusCode int, errorCode, message string) Response {
	return Response{
		StatusCode: statusCode,
		Body: map[string]string{
			"error_code": errorCode,
			"message":    message,
		},
	}
}

// quotedMaterializedFeatureIDs renders an id list the way FeatureStoreMessages.quotedIds does.
func quotedMaterializedFeatureIDs(ids []string) string {
	quoted := make([]string, len(ids))
	for i, id := range ids {
		quoted[i] = "'" + id + "'"
	}
	return strings.Join(quoted, ", ")
}

// materializedFeatureFieldCannotBeChanged refuses a field the set update cannot write. No
// acceptance case reaches it: the planner classifies every member field as a recreate, so a
// declared change never becomes an update request.
func materializedFeatureFieldCannotBeChanged(
	featureName, materializedFeatureID, fieldPath string,
) Response {
	return materializationSetError(http.StatusBadRequest, "INVALID_PARAMETER_VALUE",
		fmt.Sprintf("The fields '%s' of materialized feature '%s' of feature '%s' cannot be "+
			"changed. Only removing members is supported; recreate the materialization set "+
			"instead.", fieldPath, materializedFeatureID, featureName))
}

// materializationSetMemberKey matches a requested member to a stored one the way the backend does:
// by feature name, online store and whether the trigger is streaming. Feature name alone is not
// unique because one feature has an offline member plus one member per online store.
func materializationSetMemberKey(member ml.MaterializedFeature) string {
	onlineStoreName := ""
	if member.OnlineStoreConfig != nil {
		onlineStoreName = member.OnlineStoreConfig.OnlineStoreName
	}
	return fmt.Sprintf("%s|%s|%t", member.FeatureName, onlineStoreName, member.StreamingMode != nil)
}

// effectivePipelineScheduleState resolves the create-path default, so a request mixing unset and
// ACTIVE members is not read as a disagreement.
func effectivePipelineScheduleState(
	state ml.MaterializedFeaturePipelineScheduleState,
) ml.MaterializedFeaturePipelineScheduleState {
	if state == "" {
		return ml.MaterializedFeaturePipelineScheduleStateActive
	}
	return state
}

// populateMaterializationSetMember fills the fields the backend assigns on create, then redacts
// the INPUT_ONLY destination fields the way the response path does. Storing them redacted is what
// makes a read here match a read from a real workspace, which an echoing fake hides.
func populateMaterializationSetMember(member ml.MaterializedFeature, setID string) ml.MaterializedFeature {
	member.MaterializedFeatureId = nextUUID()
	member.MaterializationSetId = setID
	member.IsOnline = member.OnlineStoreConfig != nil
	member.PipelineScheduleState = effectivePipelineScheduleState(member.PipelineScheduleState)
	member.OfflineStoreConfig = nil
	if member.OnlineStoreConfig != nil {
		member.OnlineStoreConfig = &ml.OnlineStoreConfig{
			OnlineStoreName: member.OnlineStoreConfig.OnlineStoreName,
		}
	}
	return member
}

// validateBatchPipelineScheduleStates mirrors the create-path rule that every batch member agrees
// on one effective schedule state. Streaming members never carry one, so they are excluded.
func validateBatchPipelineScheduleStates(members []ml.MaterializedFeature) *Response {
	states := map[ml.MaterializedFeaturePipelineScheduleState]bool{}
	for _, member := range members {
		if member.StreamingMode != nil {
			continue
		}
		states[effectivePipelineScheduleState(member.PipelineScheduleState)] = true
	}
	if len(states) > 1 {
		response := materializationSetError(http.StatusBadRequest, "INVALID_PARAMETER_VALUE",
			"All materialized features in a BatchCreateMaterializedFeaturesRequest call must "+
				"have the same pipeline schedule state.")
		return &response
	}
	return nil
}

// validateSingleCronScheduleMode mirrors the create-path rule that one request carries one cron
// schedule mode. A sawtooth feature must be DERIVED and a batch-only feature MANUAL, so this is
// what a set holding both runs into first.
func validateSingleCronScheduleMode(members []ml.MaterializedFeature) *Response {
	modes := map[ml.CronScheduleMode]bool{}
	for _, member := range members {
		if member.CronScheduleTrigger == nil {
			continue
		}
		modes[member.CronScheduleTrigger.Mode] = true
	}
	if len(modes) > 1 {
		response := materializationSetError(http.StatusBadRequest, "INVALID_PARAMETER_VALUE",
			"All cron-scheduled features in a single request must use the same cron schedule "+
				"mode; a request cannot mix DERIVED (service-derived) and MANUAL "+
				"(caller-supplied) schedules.")
		return &response
	}
	return nil
}

// featureMaterializationDestination renders the destination pair the create compares across
// features: a feature's offline config plus the online config it is also materialized to, if any.
func featureMaterializationDestination(members []ml.MaterializedFeature) string {
	offline, online := "", ""
	for _, member := range members {
		if cfg := member.OfflineStoreConfig; cfg != nil {
			offline = cfg.CatalogName + "." + cfg.SchemaName + "." + cfg.TableNamePrefix
		}
		if cfg := member.OnlineStoreConfig; cfg != nil {
			online = cfg.OnlineStoreName + ":" + cfg.CatalogName + "." + cfg.SchemaName + "." +
				cfg.TableNamePrefix
		}
	}
	return offline + "|" + online
}

// validateSharedMaterializationDestination mirrors the create-path rule that every feature in a
// set asks for an identical destination pair. Because the pair carries the online config, a set
// cannot mix a feature materialized to both stores with one materialized offline only.
func validateSharedMaterializationDestination(members []ml.MaterializedFeature) *materializationSetRefusal {
	membersByFeature := map[string][]ml.MaterializedFeature{}
	var featureOrder []string
	for _, member := range members {
		if _, seen := membersByFeature[member.FeatureName]; !seen {
			featureOrder = append(featureOrder, member.FeatureName)
		}
		membersByFeature[member.FeatureName] = append(membersByFeature[member.FeatureName], member)
	}

	destinations := map[string]bool{}
	for _, featureName := range featureOrder {
		destinations[featureMaterializationDestination(membersByFeature[featureName])] = true
	}
	if len(destinations) > 1 {
		return &materializationSetRefusal{
			errorCode: "BAD_REQUEST",
			message:   "All features must share the same materialization destination.",
		}
	}
	return validateOfflineAndOnlineDestinationsDiffer(members)
}

// validateOfflineAndOnlineDestinationsDiffer mirrors the create-path rule that a feature
// materialized to both stores must separate them by catalog, schema or prefix.
func validateOfflineAndOnlineDestinationsDiffer(members []ml.MaterializedFeature) *materializationSetRefusal {
	var offline *ml.OfflineStoreConfig
	var online *ml.OnlineStoreConfig
	for _, member := range members {
		if member.OfflineStoreConfig != nil {
			offline = member.OfflineStoreConfig
		}
		if member.OnlineStoreConfig != nil {
			online = member.OnlineStoreConfig
		}
	}
	if offline == nil || online == nil {
		return nil
	}
	if online.CatalogName != offline.CatalogName || online.SchemaName != offline.SchemaName ||
		online.TableNamePrefix != offline.TableNamePrefix {
		return nil
	}
	return &materializationSetRefusal{
		errorCode: "BAD_REQUEST",
		message: fmt.Sprintf("Offline and online materialization destinations must differ. Found "+
			"identical catalog names '%s', schema names '%s', and table name prefixes '%s'.",
			offline.CatalogName, offline.SchemaName, offline.TableNamePrefix),
	}
}

// sawtoothMaterializedFeatureCount is how many materialized features a sawtooth window produces:
// an offline batch one, an online batch one and an online streaming one.
const sawtoothMaterializedFeatureCount = 3

// isSawtoothFeature reports whether a stored feature aggregates over a sawtooth window. An unknown
// feature is not treated as sawtooth: the real create resolves the feature first and fails on a
// missing one, which is a separate rejection this fake does not model.
func (s *FakeWorkspace) isSawtoothFeature(featureName string) bool {
	feature, stored := s.Features[featureName]
	if !stored {
		return false
	}
	aggregation := feature.Function.AggregationFunction
	return aggregation != nil && aggregation.TimeWindow != nil && aggregation.TimeWindow.Sawtooth != nil
}

// validateBatchAndStreamingFeaturesNotMixed mirrors the create-path rule that one call materializes
// either batch features or streaming features. A Stream-sourced feature always materializes on the
// streaming path; a Delta-sourced one joins it only when a member of it carries a streaming
// trigger. An unknown feature falls in neither group, for the reason isSawtoothFeature gives.
func (s *FakeWorkspace) validateBatchAndStreamingFeaturesNotMixed(
	members []ml.MaterializedFeature,
) *materializationSetRefusal {
	var batchFeatureNames, streamingFeatureNames []string
	for _, featureName := range materializationSetFeatureNames(members) {
		feature, stored := s.Features[featureName]
		if !stored {
			continue
		}
		hasStreamingMember := false
		for _, member := range members {
			if member.FeatureName == featureName && member.StreamingMode != nil {
				hasStreamingMember = true
			}
		}
		switch {
		case feature.Source.StreamSource != nil || hasStreamingMember:
			streamingFeatureNames = append(streamingFeatureNames, featureName)
		case feature.Source.DeltaTableSource != nil:
			batchFeatureNames = append(batchFeatureNames, featureName)
		}
	}
	if len(batchFeatureNames) == 0 || len(streamingFeatureNames) == 0 {
		return nil
	}

	return &materializationSetRefusal{
		errorCode: "INVALID_PARAMETER_VALUE",
		message: fmt.Sprintf("Batch and streaming features cannot be mixed in a single "+
			"materialize_features call because they require different trigger types. Issue two "+
			"separate calls: one for batch features and one for streaming features. Batch "+
			"features: %s. Streaming features: %s",
			strings.Join(batchFeatureNames, ", "), strings.Join(streamingFeatureNames, ", ")),
	}
}

// isDerivedCronSchedule mirrors MaterializedFeatureUtils.isDerivedCronSchedule: only a cron
// schedule trigger carrying DERIVED opts out of a pinned schedule, and an unset mode reads as
// MANUAL.
func isDerivedCronSchedule(member ml.MaterializedFeature) bool {
	return member.CronScheduleTrigger != nil &&
		member.CronScheduleTrigger.Mode == ml.CronScheduleModeDerived
}

// validateSawtoothMaterializedFeatures mirrors the rule that a sawtooth feature declares all three
// of its materialized features itself. The request is not expanded for it: a materialization set
// stamps a set id on every member before the backend reaches its normalization, and that
// normalization is skipped whenever the id is set, so a member maps to exactly one materialized
// feature and the set stays diffable.
func (s *FakeWorkspace) validateSawtoothMaterializedFeatures(
	members []ml.MaterializedFeature,
) *materializationSetRefusal {
	byFeature := map[string][]ml.MaterializedFeature{}
	var featureOrder []string
	for _, member := range members {
		if !s.isSawtoothFeature(member.FeatureName) {
			continue
		}
		if _, seen := byFeature[member.FeatureName]; !seen {
			featureOrder = append(featureOrder, member.FeatureName)
		}
		byFeature[member.FeatureName] = append(byFeature[member.FeatureName], member)
	}

	for _, featureName := range featureOrder {
		var offlineBatch, onlineBatch, onlineStreaming int
		for _, member := range byFeature[featureName] {
			switch {
			case member.OnlineStoreConfig != nil && member.StreamingMode != nil:
				onlineStreaming++
			case member.OnlineStoreConfig != nil:
				onlineBatch++
			case member.OfflineStoreConfig != nil && member.StreamingMode == nil:
				offlineBatch++
			}
		}
		if len(byFeature[featureName]) != sawtoothMaterializedFeatureCount ||
			offlineBatch != 1 || onlineBatch != 1 || onlineStreaming != 1 {
			return &materializationSetRefusal{
				errorCode: "BAD_REQUEST",
				message: fmt.Sprintf(
					"Feature '%s' uses a sawtooth window, which requires an offline batch "+
						"materialization, an online batch materialization and an online streaming "+
						"materialization. Declare one materialize feature request for each.",
					featureName),
			}
		}

		for _, member := range byFeature[featureName] {
			if member.StreamingMode != nil || isDerivedCronSchedule(member) {
				continue
			}
			return &materializationSetRefusal{
				errorCode: "BAD_REQUEST",
				message: fmt.Sprintf(
					"Feature '%s' must declare a DERIVED cron schedule on its batch materialize "+
						"feature requests. The sawtooth batch materializations refresh on a daily "+
						"schedule the service computes, so a caller neither pins nor omits it.",
					featureName),
			}
		}
	}
	return nil
}

// MaterializationSetsCreate fakes POST /api/2.0/feature-engineering/materialization-sets.
func (s *FakeWorkspace) MaterializationSetsCreate(req Request) Response {
	defer s.LockUnlock()()

	var set ml.MaterializationSet
	if err := json.Unmarshal(req.Body, &set); err != nil {
		return materializationSetInternalError(err)
	}

	if len(set.MaterializedFeatures) == 0 {
		return materializationSetError(http.StatusBadRequest, "INVALID_PARAMETER_VALUE",
			"A materialization set requires at least one materialized feature.")
	}
	if bad := validateBatchPipelineScheduleStates(set.MaterializedFeatures); bad != nil {
		return *bad
	}
	if bad := validateSingleCronScheduleMode(set.MaterializedFeatures); bad != nil {
		return *bad
	}
	set.MaterializationSetId = nextUUID()

	// The operation record is reserved here, so what follows is the operation's outcome rather
	// than a rejected request: the backend reaches these rules inside the batch create it calls
	// after reserving, and reports them by settling the record.
	for _, bad := range []*materializationSetRefusal{
		s.validateBatchAndStreamingFeaturesNotMixed(set.MaterializedFeatures),
		validateSharedMaterializationDestination(set.MaterializedFeatures),
		s.validateSawtoothMaterializedFeatures(set.MaterializedFeatures),
	} {
		if bad == nil {
			continue
		}
		return s.materializationSetFailedOperationResponse(set.MaterializationSetId,
			set.MaterializedFeatures, bad.errorCode, bad.message, nil)
	}
	for i, member := range set.MaterializedFeatures {
		set.MaterializedFeatures[i] = populateMaterializationSetMember(member, set.MaterializationSetId)
	}

	s.MaterializationSets[set.MaterializationSetId] = set

	return s.materializationSetOperationResponse(set.MaterializationSetId, set.MaterializedFeatures, set)
}

// materializationSetOperationResponse records the terminal operation a set change returns and
// renders it as the endpoint's response. The work is finished by the time the response is written,
// so the operation is recorded already terminal; a caller still has to poll it, because a create's
// or update's resulting set is reachable only from there.
func (s *FakeWorkspace) materializationSetOperationResponse(
	setID string,
	members []ml.MaterializedFeature,
	result any,
) Response {
	metadata, err := json.Marshal(ml.MaterializationSetOperationMetadata{
		FeatureFullNames:     materializationSetFeatureNames(members),
		MaterializationSetId: setID,
		State:                ml.MaterializationSetOperationMetadataStateSucceeded,
	})
	if err != nil {
		return materializationSetInternalError(err)
	}
	response, err := json.Marshal(result)
	if err != nil {
		return materializationSetInternalError(err)
	}

	operation := ml.Operation{
		Name:     "operations/" + nextUUID(),
		Done:     true,
		Metadata: metadata,
		Response: response,
	}
	s.Operations[operation.Name] = operation
	return Response{Body: operation}
}

// materializedFeatureTeardownFailsMarker is how a fixture asks the fake for a member the backend
// cannot tear down. A real teardown fails on infrastructure -- a job, a pipeline or a table the
// delete cannot reach -- which no request can ask for, so the trigger has to be in the fixture.
const materializedFeatureTeardownFailsMarker = "teardown_fails"

// materializationSetTeardownFailures reports the members of `targets` the fake refuses to tear
// down, one failure each, in target order.
func materializationSetTeardownFailures(
	targets [][]ml.MaterializedFeature,
	removedIDs map[string]bool,
) []ml.MaterializedFeatureChangeFailure {
	var failures []ml.MaterializedFeatureChangeFailure
	for _, target := range targets {
		if !strings.Contains(target[0].FeatureName, materializedFeatureTeardownFailsMarker) {
			continue
		}
		for _, member := range target {
			if !removedIDs[member.MaterializedFeatureId] {
				continue
			}
			failures = append(failures, ml.MaterializedFeatureChangeFailure{
				MaterializedFeatureId: member.MaterializedFeatureId,
				ErrorCode:             "INTERNAL_ERROR",
				Message: fmt.Sprintf(
					"Failed to delete materialized feature '%s'. If the issue persists, please "+
						"retry or contact support.", member.MaterializedFeatureId),
			})
		}
	}
	return failures
}

// partialMaterializationSetChangeErrorCode collapses the member failures to one code the way the
// backend does: a single code is reported as itself, a permission failure wins over any other, and
// a mix of client errors becomes BAD_REQUEST.
func partialMaterializationSetChangeErrorCode(
	failures []ml.MaterializedFeatureChangeFailure,
) string {
	codes := map[string]bool{}
	for _, failure := range failures {
		codes[failure.ErrorCode] = true
	}
	switch {
	case len(codes) == 1:
		for code := range codes {
			return code
		}
		return "INTERNAL_ERROR"
	case codes["PERMISSION_DENIED"]:
		return "PERMISSION_DENIED"
	default:
		return "INTERNAL_ERROR"
	}
}

// materializationSetFailedOperationResponse settles an operation as FAILED and returns it with
// HTTP 200, which is how the backend reports a change it reserved an operation for and then could
// not finish. The per-member outcome rides on the metadata's progress arm.
func (s *FakeWorkspace) materializationSetFailedOperationResponse(
	setID string,
	members []ml.MaterializedFeature,
	errorCode string,
	message string,
	progress func(*ml.MaterializationSetOperationMetadata),
) Response {
	settled := ml.MaterializationSetOperationMetadata{
		FeatureFullNames:     materializationSetFeatureNames(members),
		MaterializationSetId: setID,
		State:                ml.MaterializationSetOperationMetadataStateFailed,
	}
	if progress != nil {
		progress(&settled)
	}

	metadata, err := json.Marshal(settled)
	if err != nil {
		return materializationSetInternalError(err)
	}

	operation := ml.Operation{
		Name:     "operations/" + nextUUID(),
		Done:     true,
		Metadata: metadata,
		Error: &ml.DatabricksServiceExceptionWithDetailsProto{
			ErrorCode: ml.ErrorCode(errorCode),
			Message:   message,
		},
	}
	s.Operations[operation.Name] = operation
	return Response{Body: operation}
}

// materializationSetFeatureNames lists each member's feature once in member order, which is what
// the operation metadata reports.
func materializationSetFeatureNames(members []ml.MaterializedFeature) []string {
	seen := map[string]bool{}
	var names []string
	for _, member := range members {
		if !seen[member.FeatureName] {
			seen[member.FeatureName] = true
			names = append(names, member.FeatureName)
		}
	}
	return names
}

func materializationSetInternalError(err error) Response {
	return Response{
		Body:       fmt.Sprintf("internal error: %s", err),
		StatusCode: http.StatusInternalServerError,
	}
}

// materializationSetNotFound refuses a set id no set holds.
func materializationSetNotFound(setID string) Response {
	return materializationSetError(http.StatusNotFound, "NOT_FOUND",
		fmt.Sprintf("Materialization set with ID '%s' does not exist.", setID))
}

// MaterializationSetsDelete fakes DELETE
// /api/2.0/feature-engineering/materialization-sets/{materialization_set_id}. The delete returns a
// long-running operation too, settling with an empty response.
func (s *FakeWorkspace) MaterializationSetsDelete(setID string) Response {
	defer s.LockUnlock()()

	existing, ok := s.MaterializationSets[setID]
	if !ok {
		return materializationSetNotFound(setID)
	}

	allIDs := map[string]bool{}
	for _, member := range existing.MaterializedFeatures {
		allIDs[member.MaterializedFeatureId] = true
	}
	targets := materializationSetDeletionTargets(existing.MaterializedFeatures)
	if failures := materializationSetTeardownFailures(targets, allIDs); len(failures) > 0 {
		// The set is kept so the members that still need deleting can be retried.
		return s.materializationSetFailedOperationResponse(setID, existing.MaterializedFeatures,
			partialMaterializationSetChangeErrorCode(failures),
			fmt.Sprintf("Deleted %d of %d materialized features in materialization set '%s'. "+
				"The operation's progress lists which deletion succeeded, which failed and why.",
				0, len(failures), setID),
			func(metadata *ml.MaterializationSetOperationMetadata) {
				metadata.DeletionProgress = &ml.MaterializationSetDeletionProgress{
					FailedMaterializedFeatures: failures,
				}
			})
	}

	delete(s.MaterializationSets, setID)

	return s.materializationSetOperationResponse(setID, existing.MaterializedFeatures, struct{}{})
}

// OperationsGet fakes GET /api/2.0/feature-engineering/operations/{operation_id}.
func (s *FakeWorkspace) OperationsGet(operationID string) Response {
	defer s.LockUnlock()()

	operation, ok := s.Operations["operations/"+operationID]
	if !ok {
		// Unlike every other message here, this wording is not taken from the backend: an unknown
		// operation is resolved inside the entity-store client rather than from
		// FeatureStoreMessages. No case reaches this branch, so nothing pins it either way.
		return materializationSetError(http.StatusNotFound, "NOT_FOUND",
			fmt.Sprintf("Operation with name 'operations/%s' does not exist.", operationID))
	}
	return Response{Body: operation}
}

// effectiveCronScheduleMode treats an unset mode as MANUAL, so spelling MANUAL out is no change.
func effectiveCronScheduleMode(mode ml.CronScheduleMode) ml.CronScheduleMode {
	if mode == "" {
		return ml.CronScheduleModeManual
	}
	return mode
}

// isCronScheduleChangeRequested mirrors the backend's comparison. The mode is always compared, so
// dropping DERIVED reads as asking to stop deriving; a DERIVED schedule's expression and timezone
// belong to the service, so leaving them out asks for nothing. That is what keeps a sawtooth
// feature updatable, where comparing the field as a whole would reject every update after create.
func isCronScheduleChangeRequested(requested, existing *ml.CronSchedule) bool {
	if requested == nil || existing == nil {
		return false
	}
	requestedMode := effectiveCronScheduleMode(requested.Mode)
	if requestedMode != effectiveCronScheduleMode(existing.Mode) {
		return true
	}
	if requestedMode == ml.CronScheduleModeDerived {
		if requested.CronExpression != "" && requested.CronExpression != existing.CronExpression {
			return true
		}
	} else if requested.CronExpression != existing.CronExpression {
		return true
	}
	return requested.TimezoneId != "" && requested.TimezoneId != existing.TimezoneId
}

// materializationSetDeletionTargets groups each feature's members into the one a delete is applied
// to and the ones it cascades to. Deleting a feature's offline member deletes its online ones; a
// feature with no offline member is deleted through each online one.
func materializationSetDeletionTargets(members []ml.MaterializedFeature) [][]ml.MaterializedFeature {
	byFeature := map[string][]ml.MaterializedFeature{}
	var featureOrder []string
	for _, member := range members {
		if _, seen := byFeature[member.FeatureName]; !seen {
			featureOrder = append(featureOrder, member.FeatureName)
		}
		byFeature[member.FeatureName] = append(byFeature[member.FeatureName], member)
	}

	var targets [][]ml.MaterializedFeature
	for _, featureName := range featureOrder {
		var offline, online []ml.MaterializedFeature
		for _, member := range byFeature[featureName] {
			if member.IsOnline {
				online = append(online, member)
			} else {
				offline = append(offline, member)
			}
		}
		if len(offline) == 0 {
			for _, member := range online {
				targets = append(targets, []ml.MaterializedFeature{member})
			}
			continue
		}
		// Assumes one offline destination per feature, as the backend does.
		targets = append(targets, append([]ml.MaterializedFeature{offline[0]}, online...))
	}
	return targets
}

// MaterializationSetsUpdate fakes PATCH
// /api/2.0/feature-engineering/materialization-sets/{materialization_set_id}. Field updates are
// validated before any deletion, so the destructive half is never attempted on a set that could
// not be fully updated.
func (s *FakeWorkspace) MaterializationSetsUpdate(req Request, setID string) Response {
	defer s.LockUnlock()()

	existing, ok := s.MaterializationSets[setID]
	if !ok {
		return materializationSetNotFound(setID)
	}

	var requested ml.MaterializationSet
	if err := json.Unmarshal(req.Body, &requested); err != nil {
		return materializationSetInternalError(err)
	}

	if len(requested.MaterializedFeatures) == 0 {
		return materializationSetError(http.StatusBadRequest, "INVALID_PARAMETER_VALUE",
			fmt.Sprintf("Updating materialization set '%s' with no materialized features is not "+
				"supported. Delete the set instead.", setID))
	}

	storedByKey := map[string]ml.MaterializedFeature{}
	for _, member := range existing.MaterializedFeatures {
		storedByKey[materializationSetMemberKey(member)] = member
	}

	// Match each requested member to a stored one. A requested member matching nothing is an
	// addition, which this endpoint does not support.
	matched := make([]ml.MaterializedFeature, 0, len(requested.MaterializedFeatures))
	requestedKeys := map[string]bool{}
	for _, member := range requested.MaterializedFeatures {
		stored, found := storedByKey[materializationSetMemberKey(member)]
		if !found {
			return materializationSetError(http.StatusBadRequest, "INVALID_PARAMETER_VALUE",
				fmt.Sprintf("Materialized features cannot be added to an existing materialization "+
					"set, and feature '%s' does not match any member of this set. Recreate the "+
					"set instead.", member.FeatureName))
		}
		if isCronScheduleChangeRequested(member.CronScheduleTrigger, stored.CronScheduleTrigger) {
			return materializedFeatureFieldCannotBeChanged(member.FeatureName,
				stored.MaterializedFeatureId, "trigger")
		}
		// An unset state asks for no change: the server falls back to the stored one before
		// comparing, so a member the config never declares is not read as a change.
		if member.PipelineScheduleState != "" &&
			member.PipelineScheduleState != stored.PipelineScheduleState {
			return materializedFeatureFieldCannotBeChanged(member.FeatureName,
				stored.MaterializedFeatureId, "pipeline_schedule_state")
		}
		requestedKeys[materializationSetMemberKey(member)] = true
		matched = append(matched, stored)
	}

	// Delete the members the request omits. Because a delete cascades from a feature's offline
	// member to its online ones, a removal that would also reach a member the request still lists
	// cannot be expressed and is refused before anything is written. The server groups the removed
	// members by feature and names that feature's removed ids, so the message is built the same way.
	removedIDs := map[string]bool{}
	featureNameToRemovedIDs := map[string][]string{}
	for _, member := range existing.MaterializedFeatures {
		if !requestedKeys[materializationSetMemberKey(member)] {
			removedIDs[member.MaterializedFeatureId] = true
			featureNameToRemovedIDs[member.FeatureName] = append(
				featureNameToRemovedIDs[member.FeatureName], member.MaterializedFeatureId)
		}
	}
	for _, target := range materializationSetDeletionTargets(existing.MaterializedFeatures) {
		var unrequested []string
		requestedForChange := false
		for _, member := range target {
			if removedIDs[member.MaterializedFeatureId] {
				requestedForChange = true
			} else {
				unrequested = append(unrequested, member.MaterializedFeatureId)
			}
		}
		// A target the caller asked nothing about is not in play.
		if !requestedForChange || len(unrequested) == 0 {
			continue
		}
		featureName := target[0].FeatureName
		removedForFeature := append([]string(nil), featureNameToRemovedIDs[featureName]...)
		slices.Sort(removedForFeature)
		// The server has a message for each direction. target[0] is the member a delete is applied
		// to, so naming it means the delete reaches past what the request asked for, and omitting
		// it means the named members have no delete of their own.
		if removedIDs[target[0].MaterializedFeatureId] {
			return materializationSetError(http.StatusBadRequest, "INVALID_PARAMETER_VALUE",
				fmt.Sprintf("Deleting materialized features %s of feature '%s' also deletes "+
					"materialized features %s, which the request does not delete. Delete the "+
					"unrequested materialized features as well.",
					quotedMaterializedFeatureIDs(removedForFeature), featureName,
					quotedMaterializedFeatureIDs(unrequested)))
		}
		return materializationSetError(http.StatusBadRequest, "INVALID_PARAMETER_VALUE",
			fmt.Sprintf("Materialized features %s of feature '%s' are deleted only by deleting "+
				"materialized features %s, which the request does not supply. Remove those as "+
				"well.",
				quotedMaterializedFeatureIDs(removedForFeature), featureName,
				quotedMaterializedFeatureIDs(unrequested)))
	}

	targets := materializationSetDeletionTargets(existing.MaterializedFeatures)
	if failures := materializationSetTeardownFailures(targets, removedIDs); len(failures) > 0 {
		// Nothing is written: the one target this fake can fail is the only target, so no member
		// of the change got through.
		return s.materializationSetFailedOperationResponse(setID, existing.MaterializedFeatures,
			partialMaterializationSetChangeErrorCode(failures),
			fmt.Sprintf("Changed %d of %d materialized features in materialization set '%s'. "+
				"The operation's progress lists which update succeeded, which failed and why.",
				0, len(failures), setID),
			func(metadata *ml.MaterializationSetOperationMetadata) {
				metadata.UpdateProgress = &ml.MaterializationSetUpdateProgress{
					FailedMaterializedFeatures: failures,
				}
			})
	}

	existing.MaterializedFeatures = matched
	s.MaterializationSets[setID] = existing

	return s.materializationSetOperationResponse(setID, existing.MaterializedFeatures, existing)
}

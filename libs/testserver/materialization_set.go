package testserver

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/databricks/databricks-sdk-go/service/ml"
)

// This fake mirrors UpdateMaterializationSet closely enough that one set of acceptance goldens
// serves both the local run and a run against a real workspace. Where it rejects, it returns the
// backend's own message text, so a golden recorded locally is a claim about the real API that the
// cloud run then checks.

func materializationSetError(statusCode int, errorCode, message string) Response {
	return Response{
		StatusCode: statusCode,
		Body: map[string]string{
			"error_code": errorCode,
			"message":    message,
		},
	}
}

// materializedFeatureFieldCannotBeChanged refuses a field the set update cannot write. Every
// member field is one, so the only change the endpoint applies is a removal.
func materializedFeatureFieldCannotBeChanged(featureName, fieldPath string) Response {
	return materializationSetError(http.StatusBadRequest, "INVALID_PARAMETER_VALUE",
		fmt.Sprintf("The '%s' of feature '%s' cannot be changed after its materialized features "+
			"are created. Recreate the materialization set instead.", fieldPath, featureName))
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
func validateSharedMaterializationDestination(members []ml.MaterializedFeature) *Response {
	membersByFeature := map[string][]ml.MaterializedFeature{}
	featureOrder := []string{}
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
		response := materializationSetError(http.StatusBadRequest, "BAD_REQUEST",
			"All features must share the same materialization destination.")
		return &response
	}
	return validateOfflineAndOnlineDestinationsDiffer(members)
}

// validateOfflineAndOnlineDestinationsDiffer mirrors the create-path rule that a feature
// materialized to both stores must separate them by catalog, schema or prefix.
func validateOfflineAndOnlineDestinationsDiffer(members []ml.MaterializedFeature) *Response {
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
	response := materializationSetError(http.StatusBadRequest, "BAD_REQUEST",
		fmt.Sprintf("Offline and online materialization destinations must differ. Found "+
			"identical catalog names '%s', schema names '%s', and table name prefixes '%s'.",
			offline.CatalogName, offline.SchemaName, offline.TableNamePrefix))
	return &response
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

// validateSawtoothMaterializedFeatures mirrors the rule that a sawtooth feature declares all three
// of its materialized features itself. The request is not expanded for it: a materialization set
// stamps a set id on every member before the backend reaches its normalization, and that
// normalization is skipped whenever the id is set, so a member maps to exactly one materialized
// feature and the set stays diffable.
func (s *FakeWorkspace) validateSawtoothMaterializedFeatures(members []ml.MaterializedFeature) *Response {
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
			response := materializationSetError(http.StatusBadRequest, "BAD_REQUEST", fmt.Sprintf(
				"Feature '%s' uses a sawtooth window, which requires an offline batch "+
					"materialization, an online batch materialization and an online streaming "+
					"materialization. Declare one materialize feature request for each.", featureName))
			return &response
		}
	}
	return nil
}

// MaterializationSetsCreate fakes POST /api/2.0/feature-engineering/materialization-sets.
func (s *FakeWorkspace) MaterializationSetsCreate(req Request) Response {
	defer s.LockUnlock()()

	var set ml.MaterializationSet
	if err := json.Unmarshal(req.Body, &set); err != nil {
		return Response{
			Body:       fmt.Sprintf("internal error: %s", err),
			StatusCode: http.StatusInternalServerError,
		}
	}

	if len(set.MaterializedFeatures) == 0 {
		return materializationSetError(http.StatusBadRequest, "INVALID_PARAMETER_VALUE",
			"A materialization set requires at least one materialized feature.")
	}
	if bad := validateBatchPipelineScheduleStates(set.MaterializedFeatures); bad != nil {
		return *bad
	}
	if bad := validateSharedMaterializationDestination(set.MaterializedFeatures); bad != nil {
		return *bad
	}
	if bad := s.validateSawtoothMaterializedFeatures(set.MaterializedFeatures); bad != nil {
		return *bad
	}

	set.MaterializationSetId = nextUUID()
	for i, member := range set.MaterializedFeatures {
		set.MaterializedFeatures[i] = populateMaterializationSetMember(member, set.MaterializationSetId)
	}

	s.MaterializationSets[set.MaterializationSetId] = set

	operation, err := s.recordMaterializationSetCreation(set)
	if err != nil {
		return Response{
			Body:       fmt.Sprintf("internal error: %s", err),
			StatusCode: http.StatusInternalServerError,
		}
	}
	return Response{Body: operation}
}

// recordMaterializationSetCreation stores the operation the create returns and hands back a copy.
// The work is finished by the time the response is written, so the operation is recorded already
// terminal; a caller still has to poll it, because the created set is reachable only from there.
func (s *FakeWorkspace) recordMaterializationSetCreation(set ml.MaterializationSet) (ml.Operation, error) {
	featureNames := map[string]bool{}
	var featureOrder []string
	for _, member := range set.MaterializedFeatures {
		if !featureNames[member.FeatureName] {
			featureNames[member.FeatureName] = true
			featureOrder = append(featureOrder, member.FeatureName)
		}
	}

	metadata, err := json.Marshal(ml.MaterializationSetCreationMetadata{
		FeatureFullNames:     featureOrder,
		MaterializationSetId: set.MaterializationSetId,
		State:                ml.MaterializationSetCreationMetadataStateSucceeded,
	})
	if err != nil {
		return ml.Operation{}, err
	}
	response, err := json.Marshal(set)
	if err != nil {
		return ml.Operation{}, err
	}

	operation := ml.Operation{
		Name:     "operations/" + nextUUID(),
		Done:     true,
		Metadata: metadata,
		Response: response,
	}
	s.Operations[operation.Name] = operation
	return operation, nil
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
		return materializationSetError(http.StatusNotFound, "NOT_FOUND",
			fmt.Sprintf("Materialization set with ID '%s' does not exist.", setID))
	}

	var requested ml.MaterializationSet
	if err := json.Unmarshal(req.Body, &requested); err != nil {
		return Response{
			Body:       fmt.Sprintf("internal error: %s", err),
			StatusCode: http.StatusInternalServerError,
		}
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
				"cron_schedule_trigger")
		}
		// An unset state asks for no change, so the comparison is only against a stated one.
		if member.PipelineScheduleState != "" &&
			member.PipelineScheduleState != stored.PipelineScheduleState {
			return materializedFeatureFieldCannotBeChanged(member.FeatureName,
				"pipeline_schedule_state")
		}
		requestedKeys[materializationSetMemberKey(member)] = true
		matched = append(matched, stored)
	}

	// Delete the members the request omits. Because a delete cascades from a feature's offline
	// member to its online ones, a removal that would also reach a member the request still lists
	// cannot be expressed and is refused before anything is written.
	removedIDs := map[string]bool{}
	for _, member := range existing.MaterializedFeatures {
		if !requestedKeys[materializationSetMemberKey(member)] {
			removedIDs[member.MaterializedFeatureId] = true
		}
	}
	if len(removedIDs) > 0 {
		for _, target := range materializationSetDeletionTargets(existing.MaterializedFeatures) {
			var unrequested []string
			requestedForChange := false
			for _, member := range target {
				if removedIDs[member.MaterializedFeatureId] {
					requestedForChange = true
				} else {
					unrequested = append(unrequested, "'"+member.MaterializedFeatureId+"'")
				}
			}
			// A target the caller asked nothing about is not in play.
			if requestedForChange && len(unrequested) > 0 {
				return materializationSetError(http.StatusBadRequest, "INVALID_PARAMETER_VALUE",
					fmt.Sprintf("Changing one materialized feature of feature '%s' also changes "+
						"%s, which the request does not change. Apply the same change to every "+
						"materialized feature of the feature.",
						target[0].FeatureName, strings.Join(unrequested, ", ")))
			}
		}
	}

	existing.MaterializedFeatures = matched
	s.MaterializationSets[setID] = existing
	return Response{Body: existing}
}

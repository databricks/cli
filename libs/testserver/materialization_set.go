package testserver

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/databricks/databricks-sdk-go/service/ml"
)

// materializationSetMemberKey matches a requested member to a stored one the way the backend
// does: by feature name, online store and whether the member is streaming. Feature name alone is
// not unique because one feature has an offline member plus one member per online store.
func materializationSetMemberKey(member ml.MaterializedFeature) string {
	onlineStoreName := ""
	if member.OnlineStoreConfig != nil {
		onlineStoreName = member.OnlineStoreConfig.OnlineStoreName
	}
	return fmt.Sprintf("%s|%s|%t", member.FeatureName, onlineStoreName, member.StreamingMode != nil)
}

// populateMaterializationSetMember fills the fields the backend assigns on create: the member id,
// the online flag derived from the destination, and the schedule state the pipeline starts in.
func populateMaterializationSetMember(member ml.MaterializedFeature, setID string) ml.MaterializedFeature {
	member.MaterializedFeatureId = nextUUID()
	member.MaterializationSetId = setID
	member.IsOnline = member.OnlineStoreConfig != nil
	member.PipelineScheduleState = ml.MaterializedFeaturePipelineScheduleStateActive
	return member
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
		return Response{
			StatusCode: http.StatusBadRequest,
			Body:       "materialized_features must not be empty",
		}
	}

	set.MaterializationSetId = nextUUID()
	for i, member := range set.MaterializedFeatures {
		set.MaterializedFeatures[i] = populateMaterializationSetMember(member, set.MaterializationSetId)
	}

	s.MaterializationSets[set.MaterializationSetId] = set
	return Response{Body: set}
}

// MaterializationSetsUpdate fakes PATCH
// /api/2.0/feature-engineering/materialization-sets/{materialization_set_id}. Membership is
// reconciled against the request: a member the request omits is deleted, and one it adds is
// created. A member matched to a stored one keeps that member's server-assigned fields, because
// the endpoint cannot alter a member in place.
func (s *FakeWorkspace) MaterializationSetsUpdate(req Request, setID string) Response {
	defer s.LockUnlock()()

	existing, ok := s.MaterializationSets[setID]
	if !ok {
		return Response{
			StatusCode: http.StatusNotFound,
			Body:       fmt.Sprintf("materialization set %s not found", setID),
		}
	}

	var requested ml.MaterializationSet
	if err := json.Unmarshal(req.Body, &requested); err != nil {
		return Response{
			Body:       fmt.Sprintf("internal error: %s", err),
			StatusCode: http.StatusInternalServerError,
		}
	}

	storedByKey := map[string]ml.MaterializedFeature{}
	for _, member := range existing.MaterializedFeatures {
		storedByKey[materializationSetMemberKey(member)] = member
	}

	desired := requested.MaterializedFeatures
	if len(desired) == 0 {
		return Response{
			StatusCode: http.StatusBadRequest,
			Body:       "materialized_features must not be empty",
		}
	}

	members := make([]ml.MaterializedFeature, 0, len(desired))
	for _, member := range desired {
		if stored, found := storedByKey[materializationSetMemberKey(member)]; found {
			members = append(members, stored)
			continue
		}
		members = append(members, populateMaterializationSetMember(member, setID))
	}

	existing.MaterializedFeatures = members
	s.MaterializationSets[setID] = existing
	return Response{Body: existing}
}

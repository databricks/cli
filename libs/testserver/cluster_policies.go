package testserver

import (
	"encoding/json"
	"fmt"
	"slices"

	"github.com/databricks/databricks-sdk-go/service/compute"
)

// policyFamilyDefinition mimics the real backend: a policy created from a policy
// family has its definition computed from the family and returned on read, even
// though the config never sets definition. One fixed key is enough to reproduce a
// non-empty server-computed definition so tests exercise the backend_defaults
// suppression for definition (see resources.yml cluster_policies).
func policyFamilyDefinition(familyID string) string {
	return fmt.Sprintf(`{"policy_family":{"type":"fixed","value":%q}}`, familyID)
}

func (s *FakeWorkspace) ClusterPoliciesCreate(req Request) any {
	// Unmarshal into the stored (GET) type directly: CreatePolicy and Policy
	// share JSON field names, so every config field is carried over.
	var policy compute.Policy
	if err := json.Unmarshal(req.Body, &policy); err != nil {
		return Response{StatusCode: 400, Body: fmt.Sprintf("request parsing error: %s", err)}
	}

	// The backend rejects definition and policy_family_id together.
	if policy.Definition != "" && policy.PolicyFamilyId != "" {
		return Response{
			StatusCode: 400,
			Body: map[string]string{
				"error_code": "INVALID_PARAMETER_VALUE",
				"message":    "policy_family_id and definition cannot be used together",
			},
		}
	}

	if policy.Definition == "" && policy.PolicyFamilyId != "" {
		policy.Definition = policyFamilyDefinition(policy.PolicyFamilyId)
	}

	defer s.LockUnlock()()

	id := nextUUID()
	policy.PolicyId = id
	s.ClusterPolicies[id] = policy

	return Response{Body: compute.CreatePolicyResponse{PolicyId: id}}
}

func (s *FakeWorkspace) ClusterPoliciesList(req Request) any {
	defer s.LockUnlock()()

	ids := make([]string, 0, len(s.ClusterPolicies))
	for id := range s.ClusterPolicies {
		ids = append(ids, id)
	}
	slices.Sort(ids)

	policies := make([]compute.Policy, 0, len(ids))
	for _, id := range ids {
		policies = append(policies, s.ClusterPolicies[id])
	}

	return Response{Body: compute.ListPoliciesResponse{Policies: policies}}
}

func (s *FakeWorkspace) ClusterPoliciesGet(req Request, policyId string) any {
	defer s.LockUnlock()()

	policy, ok := s.ClusterPolicies[policyId]
	if !ok {
		return Response{StatusCode: 404}
	}

	return Response{Body: policy}
}

func (s *FakeWorkspace) ClusterPoliciesEdit(req Request) any {
	var request compute.EditPolicy
	if err := json.Unmarshal(req.Body, &request); err != nil {
		return Response{StatusCode: 400, Body: fmt.Sprintf("request parsing error: %s", err)}
	}

	defer s.LockUnlock()()

	policy, ok := s.ClusterPolicies[request.PolicyId]
	if !ok {
		return Response{StatusCode: 404}
	}

	// Edit is a full replace of the writable fields; server-set fields
	// (policy_id, created_at_timestamp, creator_user_name, is_default) are kept as stored.
	policy.Name = request.Name
	policy.Definition = request.Definition
	policy.Description = request.Description
	policy.Libraries = request.Libraries
	policy.MaxClustersPerUser = request.MaxClustersPerUser
	policy.PolicyFamilyDefinitionOverrides = request.PolicyFamilyDefinitionOverrides
	policy.PolicyFamilyId = request.PolicyFamilyId
	if policy.Definition == "" && policy.PolicyFamilyId != "" {
		policy.Definition = policyFamilyDefinition(policy.PolicyFamilyId)
	}
	s.ClusterPolicies[request.PolicyId] = policy

	return Response{}
}

func (s *FakeWorkspace) ClusterPoliciesDelete(req Request) any {
	var request compute.DeletePolicy
	if err := json.Unmarshal(req.Body, &request); err != nil {
		return Response{StatusCode: 400, Body: fmt.Sprintf("request parsing error: %s", err)}
	}

	defer s.LockUnlock()()

	if _, ok := s.ClusterPolicies[request.PolicyId]; !ok {
		return Response{StatusCode: 404}
	}

	delete(s.ClusterPolicies, request.PolicyId)

	return Response{}
}

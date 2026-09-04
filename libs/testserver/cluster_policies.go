package testserver

import (
	"encoding/json"
	"fmt"
	"maps"
	"reflect"
	"slices"
	"strings"

	"github.com/databricks/databricks-sdk-go/service/compute"
	"github.com/databricks/databricks-sdk-go/service/jobs"
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

// policyElement is the part of a cluster policy element the fake applies. The backend
// supports more element types, but only these two fields materialize a value:
// "fixed" elements set Value, every limiting type ("allowlist", "blocklist", "regex",
// "range", "unlimited") may carry DefaultValue. "forbidden" only rejects and is ignored here.
type policyElement struct {
	Type         string `json:"type"`
	Value        any    `json:"value"`
	DefaultValue any    `json:"defaultValue"`
}

// effectiveValue returns the value this element materializes into a spec that omits the
// attribute, or nil if it materializes none. Mirrors the backend: "fixed" applies whether
// or not the request sets apply_policy_default_values, "defaultValue" only when it does.
// Pinned against a real workspace by
// acceptance/bundle/resources/cluster_policies/policy_value_semantics.
func (e policyElement) effectiveValue(applyDefaults bool) any {
	if e.Type == "fixed" {
		return e.Value
	}
	if applyDefaults {
		return e.DefaultValue
	}
	return nil
}

// clusterPolicyValues returns the values the policy supplies, keyed by the policy's
// attribute path (e.g. "spark_version", "custom_tags.CostCenter").
func (s *FakeWorkspace) clusterPolicyValues(policyID string, applyDefaults bool) map[string]any {
	policy, ok := s.ClusterPolicies[policyID]
	if !ok {
		return nil
	}
	var elements map[string]policyElement
	if err := json.Unmarshal([]byte(policy.Definition), &elements); err != nil {
		return nil
	}
	values := make(map[string]any, len(elements))
	for path, element := range elements {
		if value := element.effectiveValue(applyDefaults); value != nil {
			values[path] = value
		}
	}
	return values
}

// applyClusterPolicy fills in attributes the request omitted from the policy attached via
// policyID. It returns the backend's validation message when a supplied value contradicts a
// "fixed" element, or "" when the spec is acceptable; callers must hold the workspace lock and
// surface a non-empty message as a 400.
//
// spec is a pointer to any struct with cluster-spec JSON tags (compute.ClusterDetails or
// compute.ClusterSpec); the policy's attribute paths are applied against its JSON shape so
// one implementation covers both.
func (s *FakeWorkspace) applyClusterPolicy(spec any, policyID string, applyDefaults bool) string {
	if policyID == "" {
		return ""
	}
	values := s.clusterPolicyValues(policyID, applyDefaults)
	if len(values) == 0 {
		return ""
	}

	// spec came from a successful decode of the request, so re-encoding it cannot fail.
	raw, err := json.Marshal(spec)
	if err != nil {
		return ""
	}
	// doc is only read, to test whether an attribute is already present; the spec itself is
	// updated from patch below, so decoded numbers are never written back.
	var doc map[string]any
	if err := json.Unmarshal(raw, &doc); err != nil {
		return ""
	}

	// Collect only the attributes the request omitted, then unmarshal just those back onto
	// the spec. Unmarshaling the whole document instead would rebuild ForceSendFields from
	// every key present, which makes fields the backend omits (e.g. the Jobs API dropping
	// apply_policy_default_values) serialize as explicit zeros.
	patch := map[string]any{}
	for _, path := range slices.Sorted(maps.Keys(values)) {
		value := values[path]
		segments := strings.Split(path, ".")
		if existing, ok := lookup(doc, segments); ok {
			// A "fixed" element enforces its value as well as supplying it; anything else
			// only supplies a default, which the request is free to override.
			if s.isFixed(policyID, path) && !reflect.DeepEqual(existing, value) {
				// Message shape copied from the real backend for a nested attribute
				// ("custom_tags, CostCenter must be ..."); the wording for a top-level
				// attribute has not been observed, so it is only approximated here.
				return fmt.Sprintf("Cluster validation error: Validation failed for %s must be %v (is %q)",
					strings.Join(segments, ", "), value, existing)
			}
			continue
		}
		setPatch(patch, segments, value)
	}
	if len(patch) == 0 {
		return ""
	}

	if raw, err = json.Marshal(patch); err == nil {
		_ = json.Unmarshal(raw, spec)
	}
	return ""
}

// isFixed reports whether the policy's element at path is a "fixed" element.
func (s *FakeWorkspace) isFixed(policyID, path string) bool {
	var elements map[string]policyElement
	if err := json.Unmarshal([]byte(s.ClusterPolicies[policyID].Definition), &elements); err != nil {
		return false
	}
	return elements[path].Type == "fixed"
}

// lookup returns the value at the given key path in doc.
func lookup(doc map[string]any, path []string) (any, bool) {
	for _, key := range path[:len(path)-1] {
		child, ok := doc[key].(map[string]any)
		if !ok {
			return nil, false
		}
		doc = child
	}
	value, ok := doc[path[len(path)-1]]
	return value, ok
}

// setPatch records value in patch at the given key path, creating intermediate maps.
func setPatch(patch map[string]any, path []string, value any) {
	for _, key := range path[:len(path)-1] {
		next, ok := patch[key].(map[string]any)
		if !ok {
			next = map[string]any{}
			patch[key] = next
		}
		patch = next
	}
	patch[path[len(path)-1]] = value
}

// applyPolicyDefaultValues reads apply_policy_default_values from a raw cluster request body.
// compute.ClusterDetails has no such field, so it cannot be read off the decoded request.
func applyPolicyDefaultValues(body []byte) bool {
	var spec compute.ClusterSpec
	if err := json.Unmarshal(body, &spec); err != nil {
		return false
	}
	return spec.ApplyPolicyDefaultValues
}

// applyJobClusterPolicies applies attached cluster policies to every cluster spec a job can
// carry. Callers must hold the workspace lock.
//
// The Pipelines API does not expand cluster policies into the stored spec: a pipeline cluster
// with a policy_id reads back exactly as authored, verified against a real workspace by
// acceptance/bundle/resources/cluster_policies/policy_no_drift_variants. So there is
// deliberately no pipeline equivalent.
func (s *FakeWorkspace) applyJobClusterPolicies(settings *jobs.JobSettings) string {
	for i := range settings.JobClusters {
		if msg := s.applyClusterSpecPolicy(settings.JobClusters[i].NewCluster); msg != "" {
			return msg
		}
	}
	for i := range settings.Tasks {
		task := &settings.Tasks[i]
		if msg := s.applyClusterSpecPolicy(task.NewCluster); msg != "" {
			return msg
		}
		if task.ForEachTask != nil {
			if msg := s.applyClusterSpecPolicy(task.ForEachTask.Task.NewCluster); msg != "" {
				return msg
			}
		}
	}
	return ""
}

func (s *FakeWorkspace) applyClusterSpecPolicy(spec *compute.ClusterSpec) string {
	if spec == nil {
		return ""
	}
	return s.applyClusterPolicy(spec, spec.PolicyId, spec.ApplyPolicyDefaultValues)
}

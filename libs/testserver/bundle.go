package testserver

import (
	"encoding/json"
	"slices"

	sdkbundle "github.com/databricks/databricks-sdk-go/service/bundle"
)

// Minimal Deployment Metadata Service (DMS) handlers under /api/2.0/bundle,
// sufficient to seed resource state and read it back: a CreateOperation upserts
// (or deletes) a deployment-level resource, and ListResources returns the
// recorded resources for a deployment.

// BundleCreateOperation records a resource operation against a deployment. A
// create/update operation upserts the deployment-level resource; a delete
// operation removes it.
func (s *FakeWorkspace) BundleCreateOperation(req Request, deploymentID string) Response {
	var op sdkbundle.Operation
	if err := json.Unmarshal(req.Body, &op); err != nil {
		return Response{StatusCode: 400, Body: map[string]string{"message": err.Error()}}
	}

	defer s.LockUnlock()()

	if s.BundleResources[deploymentID] == nil {
		s.BundleResources[deploymentID] = map[string]sdkbundle.Resource{}
	}

	if op.ActionType == sdkbundle.OperationActionTypeOperationActionTypeDelete {
		delete(s.BundleResources[deploymentID], op.ResourceKey)
	} else {
		s.BundleResources[deploymentID][op.ResourceKey] = sdkbundle.Resource{
			Name:           "deployments/" + deploymentID + "/resources/" + op.ResourceKey,
			ResourceKey:    op.ResourceKey,
			ResourceId:     op.ResourceId,
			LastActionType: op.ActionType,
			State:          op.State,
		}
	}

	return Response{Body: op}
}

// BundleListResources returns the resources recorded for a deployment, sorted by
// resource key for deterministic output.
func (s *FakeWorkspace) BundleListResources(deploymentID string) Response {
	defer s.LockUnlock()()

	resources := s.BundleResources[deploymentID]
	keys := make([]string, 0, len(resources))
	for k := range resources {
		keys = append(keys, k)
	}
	slices.Sort(keys)

	out := make([]sdkbundle.Resource, 0, len(keys))
	for _, k := range keys {
		out = append(out, resources[k])
	}

	return Response{Body: sdkbundle.ListResourcesResponse{Resources: out}}
}

package testserver

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/databricks/databricks-sdk-go/service/apps"
)

func (s *FakeWorkspace) AppSpaceUpsert(req Request, name string) Response {
	var space apps.Space
	if err := json.Unmarshal(req.Body, &space); err != nil {
		return Response{
			Body:       fmt.Sprintf("internal error: %s", err),
			StatusCode: http.StatusInternalServerError,
		}
	}

	defer s.LockUnlock()()

	if name != "" {
		// Update path
		existing, ok := s.AppSpaces[name]
		if !ok {
			return Response{StatusCode: 404}
		}
		if space.Description != "" {
			existing.Description = space.Description
		}
		if space.Resources != nil {
			existing.Resources = space.Resources
		}
		if space.UserApiScopes != nil {
			existing.UserApiScopes = space.UserApiScopes
		}
		if space.UsagePolicyId != "" {
			existing.UsagePolicyId = space.UsagePolicyId
		}
		s.AppSpaces[name] = existing
		space = existing
	} else {
		// Create path
		name = space.Name
		if name == "" {
			return Response{StatusCode: 400, Body: "name is required"}
		}
		if _, exists := s.AppSpaces[name]; exists {
			return Response{
				StatusCode: 409,
				Body: map[string]string{
					"error_code": "RESOURCE_ALREADY_EXISTS",
					"message":    "A space with the same name already exists: " + name,
				},
			}
		}
		space.Id = strconv.Itoa(len(s.AppSpaces) + 2000)
		space.Status = &apps.SpaceStatus{
			State: apps.SpaceStatusSpaceStateSpaceActive,
		}
		space.ServicePrincipalClientId = nextUUID()
		space.ServicePrincipalId = nextID()
		space.ServicePrincipalName = "space-" + name
		s.AppSpaces[name] = space
	}

	spaceJSON, _ := json.Marshal(space)
	return Response{
		Body: apps.Operation{
			Done:     true,
			Name:     name,
			Response: spaceJSON,
		},
	}
}

func (s *FakeWorkspace) AppSpaceGetOperation(_ Request, name string) Response {
	defer s.LockUnlock()()

	// Return a completed operation regardless of whether the space exists.
	// This supports polling after delete operations.
	space, ok := s.AppSpaces[name]
	if ok {
		spaceJSON, _ := json.Marshal(space)
		return Response{
			Body: apps.Operation{
				Done:     true,
				Name:     name,
				Response: spaceJSON,
			},
		}
	}

	emptyJSON, _ := json.Marshal(map[string]any{})
	return Response{
		Body: apps.Operation{
			Done:     true,
			Name:     name,
			Response: emptyJSON,
		},
	}
}

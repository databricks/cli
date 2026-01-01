package testserver

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/databricks/databricks-sdk-go/service/dashboards"
)

// fakeGenieSpace wraps the SDK GenieSpace with additional fields not in the response
type fakeGenieSpace struct {
	dashboards.GenieSpace
	ParentPath string `json:"parent_path,omitempty"`
}

// Generate 32 character hex string for genie space ID
func generateGenieSpaceId() (string, error) {
	randomBytes := make([]byte, 16)
	_, err := rand.Read(randomBytes)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(randomBytes), nil
}

func (s *FakeWorkspace) GenieSpaceCreate(req Request) Response {
	defer s.LockUnlock()()

	var createReq dashboards.GenieCreateSpaceRequest
	if err := json.Unmarshal(req.Body, &createReq); err != nil {
		return Response{
			StatusCode: 400,
			Body: map[string]string{
				"message": fmt.Sprintf("Failed to parse request: %s", err),
			},
		}
	}

	spaceId, err := generateGenieSpaceId()
	if err != nil {
		return Response{
			StatusCode: 500,
			Body: map[string]string{
				"message": "Failed to generate genie space ID",
			},
		}
	}

	// Remove /Workspace prefix from parent_path. This matches the remote behavior.
	parentPath := createReq.ParentPath
	if strings.HasPrefix(parentPath, "/Workspace/") {
		parentPath = strings.TrimPrefix(parentPath, "/Workspace")
	}

	genieSpace := fakeGenieSpace{
		GenieSpace: dashboards.GenieSpace{
			SpaceId:         spaceId,
			Title:           createReq.Title,
			Description:     createReq.Description,
			WarehouseId:     createReq.WarehouseId,
			SerializedSpace: createReq.SerializedSpace,
		},
		ParentPath: parentPath,
	}

	s.GenieSpaces[spaceId] = genieSpace

	return Response{
		Body: genieSpace,
	}
}

func (s *FakeWorkspace) GenieSpaceUpdate(req Request, spaceId string) Response {
	defer s.LockUnlock()()

	genieSpace, ok := s.GenieSpaces[spaceId]
	if !ok {
		return Response{
			StatusCode: 404,
			Body: map[string]string{
				"message": fmt.Sprintf("Genie space with ID %s not found", spaceId),
			},
		}
	}

	var updateReq dashboards.GenieUpdateSpaceRequest
	if err := json.Unmarshal(req.Body, &updateReq); err != nil {
		return Response{
			StatusCode: 400,
			Body: map[string]string{
				"message": fmt.Sprintf("Failed to parse request: %s", err),
			},
		}
	}

	// Update fields if provided
	if updateReq.Title != "" {
		genieSpace.Title = updateReq.Title
	}
	if updateReq.Description != "" {
		genieSpace.Description = updateReq.Description
	}
	if updateReq.WarehouseId != "" {
		genieSpace.WarehouseId = updateReq.WarehouseId
	}
	if updateReq.SerializedSpace != "" {
		genieSpace.SerializedSpace = updateReq.SerializedSpace
	}

	s.GenieSpaces[spaceId] = genieSpace

	return Response{
		Body: genieSpace,
	}
}

func (s *FakeWorkspace) GenieSpaceTrash(spaceId string) Response {
	defer s.LockUnlock()()

	_, ok := s.GenieSpaces[spaceId]
	if !ok {
		return Response{
			StatusCode: 404,
			Body: map[string]string{
				"message": fmt.Sprintf("Genie space with ID %s not found", spaceId),
			},
		}
	}

	delete(s.GenieSpaces, spaceId)

	return Response{
		Body: map[string]any{},
	}
}

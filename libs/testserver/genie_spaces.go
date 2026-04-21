package testserver

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"path"
	"strings"

	"github.com/databricks/databricks-sdk-go/service/dashboards"
	"github.com/databricks/databricks-sdk-go/service/workspace"
)

// generateGenieSpaceId returns a random 32-character hex string.
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
				"message": "Invalid request body: " + err.Error(),
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

	genieSpace := dashboards.GenieSpace{
		SpaceId:         spaceId,
		Title:           createReq.Title,
		Description:     createReq.Description,
		WarehouseId:     createReq.WarehouseId,
		SerializedSpace: createReq.SerializedSpace,
	}

	s.GenieSpaces[spaceId] = genieSpace

	// Register in the workspace files for path lookup.
	if createReq.ParentPath != "" {
		workspacePath := createReq.ParentPath
		if !strings.HasPrefix(workspacePath, "/Workspace") {
			workspacePath = path.Join("/Workspace", workspacePath)
		}
		workspacePath = path.Join(workspacePath, createReq.Title+".genie")

		s.files[workspacePath] = FileEntry{
			Info: workspace.ObjectInfo{
				ObjectType: "FILE",
				Path:       workspacePath,
				ResourceId: spaceId,
			},
			Data: []byte(createReq.SerializedSpace),
		}
	}

	return Response{
		Body: genieSpace,
	}
}

func (s *FakeWorkspace) GenieSpaceGet(req Request) Response {
	defer s.LockUnlock()()

	spaceId := req.Vars["space_id"]
	genieSpace, ok := s.GenieSpaces[spaceId]
	if !ok {
		return Response{
			StatusCode: 404,
			Body: map[string]string{
				"message": "Genie space not found",
			},
		}
	}

	return Response{
		Body: genieSpace,
	}
}

func (s *FakeWorkspace) GenieSpaceUpdate(req Request) Response {
	defer s.LockUnlock()()

	spaceId := req.Vars["space_id"]
	genieSpace, ok := s.GenieSpaces[spaceId]
	if !ok {
		return Response{
			StatusCode: 404,
			Body: map[string]string{
				"message": "Genie space not found",
			},
		}
	}

	var updateReq dashboards.GenieUpdateSpaceRequest
	if err := json.Unmarshal(req.Body, &updateReq); err != nil {
		return Response{
			StatusCode: 400,
			Body: map[string]string{
				"message": "Invalid request body: " + err.Error(),
			},
		}
	}

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

func (s *FakeWorkspace) GenieSpaceTrash(req Request) Response {
	defer s.LockUnlock()()

	spaceId := req.Vars["space_id"]
	_, ok := s.GenieSpaces[spaceId]
	if !ok {
		return Response{
			StatusCode: 404,
			Body: map[string]string{
				"message": "Genie space not found",
			},
		}
	}

	delete(s.GenieSpaces, spaceId)

	return Response{
		StatusCode: 200,
	}
}

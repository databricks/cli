package testserver

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"path"
	"strconv"
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

	// Strip the /Workspace prefix from parent_path before storing. This matches
	// the remote behavior: the GET API returns parent_path without the prefix,
	// mirroring DashboardCreate.
	if strings.HasPrefix(createReq.ParentPath, "/Workspace/") {
		createReq.ParentPath = strings.TrimPrefix(createReq.ParentPath, "/Workspace")
	}

	genieSpace := dashboards.GenieSpace{
		SpaceId:         spaceId,
		Title:           createReq.Title,
		Description:     createReq.Description,
		ParentPath:      createReq.ParentPath,
		WarehouseId:     createReq.WarehouseId,
		SerializedSpace: createReq.SerializedSpace,
		// Mirror libs/testserver/dashboards.go: initialize etag to a numeric
		// string so each subsequent update can bump it monotonically.
		Etag: "1",
	}

	s.GenieSpaces[spaceId] = genieSpace

	// Register in the workspace files for path lookup.
	if createReq.ParentPath != "" {
		workspacePath := createReq.ParentPath
		if !strings.HasPrefix(workspacePath, "/Workspace") {
			workspacePath = path.Join("/Workspace", workspacePath)
		}
		workspacePath = path.Join(workspacePath, createReq.Title+".geniespace")

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

	// Optimistic concurrency: if the caller sent an etag, it must match the
	// current one. Empty etag means apply unconditionally.
	if updateReq.Etag != "" && updateReq.Etag != genieSpace.Etag {
		return Response{
			StatusCode: 409,
			Body: map[string]string{
				"message": "Etag mismatch: expected " + genieSpace.Etag + ", got " + updateReq.Etag,
			},
		}
	}

	prev := genieSpace
	if updateReq.Title != "" {
		genieSpace.Title = updateReq.Title
	}
	if updateReq.Description != "" {
		genieSpace.Description = updateReq.Description
	}
	if updateReq.WarehouseId != "" {
		genieSpace.WarehouseId = updateReq.WarehouseId
	}
	if updateReq.ParentPath != "" {
		parentPath := updateReq.ParentPath
		if strings.HasPrefix(parentPath, "/Workspace/") {
			parentPath = strings.TrimPrefix(parentPath, "/Workspace")
		}
		genieSpace.ParentPath = parentPath
	}
	if updateReq.SerializedSpace != "" {
		genieSpace.SerializedSpace = updateReq.SerializedSpace
	}

	// Bump the etag only when the update actually changes user-visible state.
	// Matches dashboard's behavior (libs/testserver/dashboards.go) and keeps
	// no-op updates idempotent so TestAll can pass the same config to Create
	// and Update without observing spurious drift.
	if prev.Title != genieSpace.Title ||
		prev.Description != genieSpace.Description ||
		prev.WarehouseId != genieSpace.WarehouseId ||
		prev.ParentPath != genieSpace.ParentPath ||
		prev.SerializedSpace != genieSpace.SerializedSpace {
		prevEtag, err := strconv.Atoi(genieSpace.Etag)
		if err != nil {
			return Response{
				StatusCode: 500,
				Body: map[string]string{
					"message": "Invalid stored etag: " + genieSpace.Etag,
				},
			}
		}
		genieSpace.Etag = strconv.Itoa(prevEtag + 1)
	}

	s.GenieSpaces[spaceId] = genieSpace

	return Response{
		Body: genieSpace,
	}
}

func (s *FakeWorkspace) GenieSpaceTrash(req Request) Response {
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

	delete(s.GenieSpaces, spaceId)

	// Also remove the synthetic workspace file entry registered by
	// GenieSpaceCreate, so a trash+recreate flow does not resolve to stale
	// state via the workspace path index.
	if genieSpace.ParentPath != "" {
		workspacePath := genieSpace.ParentPath
		if !strings.HasPrefix(workspacePath, "/Workspace") {
			workspacePath = path.Join("/Workspace", workspacePath)
		}
		workspacePath = path.Join(workspacePath, genieSpace.Title+".geniespace")
		delete(s.files, workspacePath)
	}

	return Response{
		StatusCode: 200,
	}
}

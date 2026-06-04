package testserver

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"strconv"
	"strings"

	"github.com/databricks/databricks-sdk-go/service/dashboards"
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

	// Default to user's home directory if parent_path is not provided (matches cloud behavior)
	if createReq.ParentPath == "" {
		createReq.ParentPath = "/Users/" + s.CurrentUser().UserName
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

	// Genie spaces are not exposed as workspace files ("dataRoom is not
	// user-facing"), so unlike dashboards we do not register a workspace path
	// entry — there is nothing to resolve via the Workspace API.

	return Response{
		Body: genieSpace,
	}
}

func (s *FakeWorkspace) GenieSpaceGet(req Request) Response {
	defer s.LockUnlock()()

	spaceId := req.Vars["space_id"]
	genieSpace, ok := s.GenieSpaces[spaceId]
	if !ok {
		// The real API returns 403 (not 404) when a Genie space does not exist.
		return Response{
			StatusCode: 403,
			Body: map[string]string{
				"message": "Genie space not found",
			},
		}
	}

	// The GET API only returns the etag when serialized_space is requested
	// (include_serialized_space=true). genieSpace is a copy, so clearing the
	// field here only affects the response.
	if req.URL.Query().Get("include_serialized_space") != "true" {
		genieSpace.Etag = ""
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
		// The real API returns 403 (not 404) when a Genie space does not exist.
		return Response{
			StatusCode: 403,
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

	// The backend bumps the etag only when serialized_space changes; updates to
	// other fields (title, description, warehouse_id, parent_path) leave it
	// unchanged. This mirrors the GET API, which only returns the etag
	// alongside serialized_space.
	if prev.SerializedSpace != genieSpace.SerializedSpace {
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
	if _, ok := s.GenieSpaces[spaceId]; !ok {
		// The real API returns 403 (not 404) when a Genie space does not exist.
		return Response{
			StatusCode: 403,
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

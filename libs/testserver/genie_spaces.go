package testserver

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"strconv"
	"strings"

	genie "github.com/databricks/cli/libs/sdktypes/genie/v1"
	"github.com/databricks/cli/libs/utils"
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

// canonicalizeGenieSerializedSpace mimics the backend's schema migration: a
// serialized_space authored at version 1 is upgraded to version 2 (gaining an
// empty data_sources object) when written. Observed on aws-prod-ucws (2026-06):
// a create with {"version":1} reads back from GET as
// {"version":2,"data_sources":{}}. The local config keeps version 1, so the
// remote-vs-local serialized_space differs permanently; the resource relies on
// ignore_remote_changes (etag_based) to avoid spurious drift. Reproducing the
// migration here lets that handling be exercised against the mock server.
func canonicalizeGenieSerializedSpace(serialized string) string {
	if serialized == "" {
		return serialized
	}
	var m map[string]any
	if err := json.Unmarshal([]byte(serialized), &m); err != nil {
		// Not a JSON object we recognize; store verbatim.
		return serialized
	}
	if v, ok := m["version"].(float64); !ok || v != 1 {
		return serialized
	}
	m["version"] = 2
	if _, ok := m["data_sources"]; !ok {
		m["data_sources"] = map[string]any{}
	}
	b, err := json.Marshal(m)
	if err != nil {
		return serialized
	}
	return string(b)
}

func (s *FakeWorkspace) GenieSpaceCreate(req Request) Response {
	defer s.LockUnlock()()

	var createReq genie.GenieCreateSpaceRequest
	if err := json.Unmarshal(req.Body, &createReq); err != nil {
		return Response{
			StatusCode: 400,
			Body: map[string]string{
				"message": "Invalid request body: " + err.Error(),
			},
		}
	}

	// Default to user's home directory if parent_path is not provided (matches cloud behavior)
	if utils.Deref(createReq.ParentPath) == "" {
		createReq.ParentPath = utils.NonZero("/Users/" + utils.Deref(s.CurrentUser().UserName))
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
	if strings.HasPrefix(utils.Deref(createReq.ParentPath), "/Workspace/") {
		createReq.ParentPath = utils.NonZero(strings.TrimPrefix(utils.Deref(createReq.ParentPath), "/Workspace"))
	}

	genieSpace := genie.GenieSpace{
		SpaceId:         utils.NonZero(spaceId),
		Title:           createReq.Title,
		Description:     createReq.Description,
		ParentPath:      createReq.ParentPath,
		WarehouseId:     createReq.WarehouseId,
		SerializedSpace: utils.NonZero(canonicalizeGenieSerializedSpace(utils.Deref(createReq.SerializedSpace))),
		// Mirror libs/testserver/dashboards.go: initialize etag to a numeric
		// string so each subsequent update can bump it monotonically.
		Etag: utils.NonZero("1"),
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
		return Response{
			StatusCode: 404,
			Body: map[string]string{
				"message": "Genie space not found",
			},
		}
	}

	// The GET API only returns the etag when serialized_space is requested
	// (include_serialized_space=true). genieSpace is a copy, so clearing the
	// field here only affects the response.
	if req.URL.Query().Get("include_serialized_space") != "true" {
		genieSpace.Etag = nil
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

	var updateReq genie.GenieUpdateSpaceRequest
	if err := json.Unmarshal(req.Body, &updateReq); err != nil {
		return Response{
			StatusCode: 400,
			Body: map[string]string{
				"message": "Invalid request body: " + err.Error(),
			},
		}
	}

	// Which fields the caller actually sent, so an explicit "description": "" (the CLI
	// force-sends it) clears the value instead of being treated as "not provided".
	fields, _ := parseUpdateFields(req.Body)

	// Optimistic concurrency: if the caller sent an etag, it must match the
	// current one. Empty etag means apply unconditionally.
	if utils.Deref(updateReq.Etag) != "" && utils.Deref(updateReq.Etag) != utils.Deref(genieSpace.Etag) {
		return Response{
			StatusCode: 409,
			Body: map[string]string{
				"message": "Etag mismatch: expected " + utils.Deref(genieSpace.Etag) + ", got " + utils.Deref(updateReq.Etag),
			},
		}
	}

	prev := genieSpace
	if utils.Deref(updateReq.Title) != "" {
		genieSpace.Title = updateReq.Title
	}
	if _, ok := fields["description"]; ok {
		genieSpace.Description = updateReq.Description
	}
	if utils.Deref(updateReq.WarehouseId) != "" {
		genieSpace.WarehouseId = updateReq.WarehouseId
	}
	if utils.Deref(updateReq.ParentPath) != "" {
		parentPath := updateReq.ParentPath
		if strings.HasPrefix(utils.Deref(parentPath), "/Workspace/") {
			parentPath = utils.NonZero(strings.TrimPrefix(utils.Deref(parentPath), "/Workspace"))
		}
		genieSpace.ParentPath = parentPath
	}
	if utils.Deref(updateReq.SerializedSpace) != "" {
		genieSpace.SerializedSpace = utils.NonZero(canonicalizeGenieSerializedSpace(utils.Deref(updateReq.SerializedSpace)))
	}

	// The backend bumps the etag only when serialized_space changes; updates to
	// other fields (title, description, warehouse_id, parent_path) leave it
	// unchanged. This mirrors the GET API, which only returns the etag
	// alongside serialized_space.
	if utils.Deref(prev.SerializedSpace) != utils.Deref(genieSpace.SerializedSpace) {
		prevEtag, err := strconv.Atoi(utils.Deref(genieSpace.Etag))
		if err != nil {
			return Response{
				StatusCode: 500,
				Body: map[string]string{
					"message": "Invalid stored etag: " + utils.Deref(genieSpace.Etag),
				},
			}
		}
		genieSpace.Etag = utils.NonZero(strconv.Itoa(prevEtag + 1))
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

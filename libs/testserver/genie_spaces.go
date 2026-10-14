package testserver

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	genie "github.com/databricks/cli/libs/sdktypes/genie/v1"
	"strconv"
	"strings"
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
	if val(createReq.ParentPath) == "" {
		createReq.ParentPath = nonZero("/Users/" + val(s.CurrentUser().UserName))
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
	if strings.HasPrefix(val(createReq.ParentPath), "/Workspace/") {
		createReq.ParentPath = nonZero(strings.TrimPrefix(val(createReq.ParentPath), "/Workspace"))
	}

	genieSpace := genie.GenieSpace{
		SpaceId:         nonZero(spaceId),
		Title:           createReq.Title,
		Description:     createReq.Description,
		ParentPath:      createReq.ParentPath,
		WarehouseId:     createReq.WarehouseId,
		SerializedSpace: nonZero(canonicalizeGenieSerializedSpace(val(createReq.SerializedSpace))),
		// Mirror libs/testserver/dashboards.go: initialize etag to a numeric
		// string so each subsequent update can bump it monotonically.
		Etag: nonZero("1"),
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
		genieSpace.Etag = nonZero("")
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
	if val(updateReq.Etag) != "" && updateReq.Etag != genieSpace.Etag {
		return Response{
			StatusCode: 409,
			Body: map[string]string{
				"message": "Etag mismatch: expected " + val(genieSpace.Etag) + ", got " + val(updateReq.Etag),
			},
		}
	}

	prev := genieSpace
	if val(updateReq.Title) != "" {
		genieSpace.Title = updateReq.Title
	}
	if _, ok := fields["description"]; ok {
		genieSpace.Description = updateReq.Description
	}
	if val(updateReq.WarehouseId) != "" {
		genieSpace.WarehouseId = updateReq.WarehouseId
	}
	if val(updateReq.ParentPath) != "" {
		parentPath := updateReq.ParentPath
		if strings.HasPrefix(val(parentPath), "/Workspace/") {
			parentPath = nonZero(strings.TrimPrefix(val(parentPath), "/Workspace"))
		}
		genieSpace.ParentPath = parentPath
	}
	if val(updateReq.SerializedSpace) != "" {
		genieSpace.SerializedSpace = nonZero(canonicalizeGenieSerializedSpace(val(updateReq.SerializedSpace)))
	}

	// The backend bumps the etag only when serialized_space changes; updates to
	// other fields (title, description, warehouse_id, parent_path) leave it
	// unchanged. This mirrors the GET API, which only returns the etag
	// alongside serialized_space.
	if prev.SerializedSpace != genieSpace.SerializedSpace {
		prevEtag, err := strconv.Atoi(val(genieSpace.Etag))
		if err != nil {
			return Response{
				StatusCode: 500,
				Body: map[string]string{
					"message": "Invalid stored etag: " + val(genieSpace.Etag),
				},
			}
		}
		genieSpace.Etag = nonZero(strconv.Itoa(prevEtag + 1))
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

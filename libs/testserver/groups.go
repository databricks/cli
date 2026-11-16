package testserver

import (
	"encoding/json"
	"fmt"
	"maps"
	"net/http"
	"slices"
	"strconv"

	"github.com/databricks/databricks-sdk-go/service/iam"
)

func (s *FakeWorkspace) GroupsCreate(req Request) Response {
	defer s.LockUnlock()()

	var request iam.Group
	if err := json.Unmarshal(req.Body, &request); err != nil {
		return Response{
			Body:       fmt.Sprintf("internal error: %s", err),
			StatusCode: 500,
		}
	}

	// Generate an ID for the group
	groupId := strconv.FormatInt(nextID(), 10)

	group := iam.Group{
		Id:          groupId,
		DisplayName: request.DisplayName,
	}

	if s.Groups == nil {
		s.Groups = make(map[string]iam.Group)
	}

	s.Groups[groupId] = group

	return Response{
		Body: group,
	}
}

// IdentityMeTransitiveParentGroups answers GET /api/2.0/identity/me/transitive-parent-groups
// with the groups the caller belongs to.
func (s *FakeWorkspace) IdentityMeTransitiveParentGroups() any {
	defer s.LockUnlock()()

	ids := slices.Sorted(maps.Keys(s.groupMembers))
	parents := make([]map[string]any, 0, len(ids))
	for _, id := range ids {
		parents = append(parents, map[string]any{"group_id": id})
	}
	return map[string]any{"me_transitive_parent_groups": parents}
}

// AddGroups creates a group for each name in member and each name in nonMember in the
// workspace behind token, and makes that identity a member of the groups in member.
func (s *Server) AddGroups(token string, member, nonMember []string) {
	ws := s.getWorkspaceForToken(token)
	defer ws.LockUnlock()()
	for _, name := range slices.Concat(member, nonMember) {
		id := strconv.FormatInt(nextID(), 10)
		ws.Groups[id] = iam.Group{Id: id, DisplayName: name}
	}
	for _, name := range member {
		for id, group := range ws.Groups {
			if group.DisplayName == name {
				ws.groupMembers[id] = true
			}
		}
	}
}

// JoinGroup makes the identity behind token a member of the group with groupID.
// The ID need not name an existing group, which lets tests make a group lookup fail.
func (s *Server) JoinGroup(token, groupID string) {
	ws := s.getWorkspaceForToken(token)
	defer ws.LockUnlock()()
	ws.groupMembers[groupID] = true
}

// IdentityGroup answers GET /api/2.0/identity/groups/{group_id}.
func (s *FakeWorkspace) IdentityGroup(groupID string) any {
	defer s.LockUnlock()()

	group, ok := s.Groups[groupID]
	if !ok {
		return Response{
			StatusCode: http.StatusNotFound,
			Body:       map[string]any{"error_code": "RESOURCE_DOES_NOT_EXIST", "message": "Group " + groupID + " does not exist."},
		}
	}
	return map[string]any{
		"name":       "groups/" + groupID,
		"group_id":   group.Id,
		"group_name": group.DisplayName,
	}
}

// IdentityGroups answers GET /api/2.0/identity/groups, one page of all groups in
// the account. page_token is the offset of the next page, as the server's opaque token.
func (s *FakeWorkspace) IdentityGroups(pageSize int, pageToken string) any {
	defer s.LockUnlock()()

	if pageSize <= 0 {
		pageSize = 1000
	}
	ids := slices.Sorted(maps.Keys(s.Groups))
	start, _ := strconv.Atoi(pageToken)
	end := min(start+pageSize, len(ids))
	groups := make([]map[string]any, 0, end-start)
	for _, id := range ids[start:end] {
		group := s.Groups[id]
		groups = append(groups, map[string]any{"group_id": group.Id, "group_name": group.DisplayName})
	}
	resp := map[string]any{"groups": groups}
	if end < len(ids) {
		resp["next_page_token"] = strconv.Itoa(end)
	}
	return resp
}

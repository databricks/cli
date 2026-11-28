package testserver

import (
	"encoding/json"
	"fmt"
	"maps"
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

	// A member that is the caller makes the caller a member of the new group, so acceptance
	// tests can set group membership declaratively.
	for _, member := range request.Members {
		if member.Value == userForToken(req.Token).Id {
			s.groupMembers[groupId] = true
		}
	}

	return Response{
		Body: group,
	}
}

// usersGroupId is the workspace-local users group. Every workspace identity is a member of it,
// so the fake lists it for every caller, as a real workspace does.
const usersGroupId = "1"

// IdentityMeTransitiveParentGroups answers GET /api/2.0/identity/me/transitive-parent-groups
// with the groups the caller belongs to, named as the server names them.
func (s *FakeWorkspace) IdentityMeTransitiveParentGroups() any {
	defer s.LockUnlock()()

	ids := slices.Sorted(maps.Keys(s.groupMembers))
	parents := make([]map[string]any, 0, len(ids)+1)
	parents = append(parents, map[string]any{"group_id": usersGroupId, "group_name": "users"})
	for _, id := range ids {
		parents = append(parents, map[string]any{"group_id": id, "group_name": s.Groups[id].DisplayName})
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

package testserver

import (
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/databricks/databricks-sdk-go/service/iam"
)

func init() {
	registerRoutes("groups", addGroupsRoutes)
	registerStore("groups", newGroupsStore)
}

type groupsStore struct {
	Groups map[string]iam.Group
}

func newGroupsStore(*FakeWorkspace) *groupsStore {
	return &groupsStore{
		Groups: map[string]iam.Group{},
	}
}

func (s *FakeWorkspace) groups() *groupsStore {
	return stateOf[groupsStore](s)
}

func addGroupsRoutes(server *Server) {
	// Groups:
	server.Handle("POST", "/api/2.0/preview/scim/v2/Groups", func(req Request) any {
		return req.Workspace.GroupsCreate(req)
	})
}

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

	if s.groups().Groups == nil {
		s.groups().Groups = make(map[string]iam.Group)
	}

	s.groups().Groups[groupId] = group

	return Response{
		Body: group,
	}
}

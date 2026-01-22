package testserver

import (
	"encoding/json"
	"fmt"
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

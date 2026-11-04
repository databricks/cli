package testserver

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/databricks/databricks-sdk-go/service/catalog"
)

func (s *FakeWorkspace) ConnectionsCreate(req Request) Response {
	defer s.LockUnlock()()

	var createRequest catalog.CreateConnection
	if err := json.Unmarshal(req.Body, &createRequest); err != nil {
		return Response{
			Body:       fmt.Sprintf("internal error: %s", err),
			StatusCode: http.StatusInternalServerError,
		}
	}

	if _, ok := s.Connections[createRequest.Name]; ok {
		return Response{
			StatusCode: http.StatusConflict,
			Body: map[string]string{
				"error_code": "CONNECTION_ALREADY_EXISTS",
				"message":    fmt.Sprintf("Connection '%s' already exists.", createRequest.Name),
			},
		}
	}

	info := catalog.ConnectionInfo{
		Name:           createRequest.Name,
		FullName:       createRequest.Name,
		ConnectionId:   nextUUID(),
		ConnectionType: createRequest.ConnectionType,
		Comment:        createRequest.Comment,
		Options:        createRequest.Options,
		ReadOnly:       createRequest.ReadOnly,
		Owner:          s.CurrentUser().UserName,
		CreatedAt:      nowMilli(),
		CreatedBy:      s.CurrentUser().UserName,
	}
	info.UpdatedAt = info.CreatedAt
	info.UpdatedBy = info.CreatedBy

	s.Connections[createRequest.Name] = info
	return Response{Body: info}
}

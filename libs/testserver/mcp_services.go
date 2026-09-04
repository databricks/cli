package testserver

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/databricks/databricks-sdk-go/service/catalog"
)

// McpServicesCreate fakes POST /api/2.1/unity-catalog/mcp-services. See
// ModelServicesCreate for the shape: the McpService body is sent directly,
// parent and mcp_service_id are query parameters, and the map is keyed by the
// {catalog}.{schema}.{mcp_service} path segment.
func (s *FakeWorkspace) McpServicesCreate(req Request) Response {
	defer s.LockUnlock()()

	var ms catalog.McpService
	if err := json.Unmarshal(req.Body, &ms); err != nil {
		return Response{
			Body:       fmt.Sprintf("internal error: %s", err),
			StatusCode: http.StatusInternalServerError,
		}
	}

	schema := strings.TrimPrefix(req.URL.Query().Get("parent"), "schemas/")
	key := schema + "." + req.URL.Query().Get("mcp_service_id")

	ms.Name = "mcp-services/" + key
	ms.CreatedBy = s.CurrentUser().UserName
	ms.UpdatedBy = s.CurrentUser().UserName
	ms.EffectiveOwner = s.CurrentUser().UserName
	ms.MetastoreId = nextUUID()

	s.McpServices[key] = ms
	return Response{
		Body: ms,
	}
}

func (s *FakeWorkspace) McpServicesUpdate(req Request, name string) Response {
	defer s.LockUnlock()()

	existing, ok := s.McpServices[name]
	if !ok {
		return Response{
			StatusCode: http.StatusNotFound,
			Body:       fmt.Sprintf("mcp service %s not found", name),
		}
	}

	var incoming catalog.McpService
	if err := json.Unmarshal(req.Body, &incoming); err != nil {
		return Response{
			Body:       fmt.Sprintf("internal error: %s", err),
			StatusCode: http.StatusInternalServerError,
		}
	}

	existing.Comment = incoming.Comment
	existing.Config = incoming.Config
	existing.UpdatedBy = s.CurrentUser().UserName

	s.McpServices[name] = existing
	return Response{
		Body: existing,
	}
}

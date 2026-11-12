package testserver

import (
	"encoding/json"
	"fmt"
	"maps"
	"net/http"
	"slices"
	"strings"

	"github.com/databricks/databricks-sdk-go/service/catalog"
)

// mcpServicesViewFull is the list view that includes source connections. Any other view,
// including unset, omits them, as the API does.
const mcpServicesViewFull = "FULL"

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

// McpServicesList fakes GET /api/2.1/unity-catalog/mcp-services. It filters by parent
// (schemas/{catalog}.{schema}) and drops the source connection unless view is FULL, as the
// real API does. Pagination is not faked: every match is returned in one page.
func (s *FakeWorkspace) McpServicesList(req Request) Response {
	defer s.LockUnlock()()

	query := req.URL.Query()
	prefix := strings.TrimPrefix(query.Get("parent"), "schemas/") + "."
	full := query.Get("view") == mcpServicesViewFull

	var services []catalog.McpService
	for _, key := range slices.Sorted(maps.Keys(s.McpServices)) {
		if !strings.HasPrefix(key, prefix) {
			continue
		}
		ms := s.McpServices[key]
		if !full && ms.Config != nil {
			// Copy so the stored service keeps its source connection for later FULL reads.
			config := *ms.Config
			config.SourceConnection = nil
			ms.Config = &config
		}
		services = append(services, ms)
	}

	return Response{
		Body: catalog.ListMcpServicesResponse{McpServices: services},
	}
}

// SeedMcpService stores an MCP service under fullName (catalog.schema.leaf) for list tests.
// An empty oauthProvider seeds an internal Databricks service with no source connection;
// otherwise the service is backed by a source connection with that OAuth provider.
func (s *FakeWorkspace) SeedMcpService(fullName, comment, oauthProvider string) {
	defer s.LockUnlock()()

	ms := catalog.McpService{
		Name:    "mcp-services/" + fullName,
		Comment: comment,
	}
	if oauthProvider != "" {
		ms.Config = &catalog.McpServiceConfig{
			SourceConnection: &catalog.McpServiceConfigSourceConnection{
				Name:    "connections/" + fullName,
				Options: map[string]string{"oauth_provider": oauthProvider},
			},
		}
	}
	s.McpServices[fullName] = ms
}

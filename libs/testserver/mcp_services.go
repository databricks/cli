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

// Resource-name prefixes of MCP services and their parent schemas.
const (
	mcpServicesNamePrefix   = "mcp-services/"
	mcpServicesParentPrefix = "schemas/"
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

	schema := strings.TrimPrefix(req.URL.Query().Get("parent"), mcpServicesParentPrefix)
	key := schema + "." + req.URL.Query().Get("mcp_service_id")

	ms.Name = mcpServicesNamePrefix + key
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
// (schemas/{catalog}.{schema}, in any casing, as Unity Catalog names are case-insensitive) and
// drops the source connection unless view is FULL, as the real API does. Pagination is not
// faked: every match is returned in one page.
func (s *FakeWorkspace) McpServicesList(req Request) Response {
	defer s.LockUnlock()()

	query := req.URL.Query()
	prefix := strings.ToLower(strings.TrimPrefix(query.Get("parent"), mcpServicesParentPrefix)) + "."
	full := query.Get("view") == string(catalog.ListMcpServicesRequestViewFull)

	var services []catalog.McpService
	for _, key := range slices.Sorted(maps.Keys(s.McpServices)) {
		if !strings.HasPrefix(strings.ToLower(key), prefix) {
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

// McpUserCredentialCreate fakes POST .../mcp-services/{name}/user-credentials, which completes a
// caller's provider sign-in. Tests use it to flip a credential from absent to ACTIVE while the CLI
// polls. The login body is not checked.
func (s *FakeWorkspace) McpUserCredentialCreate(service string) Response {
	defer s.LockUnlock()()

	cred := catalog.McpServiceUserMappedCredential{
		ProvisioningInfo: &catalog.ProvisioningInfo{State: catalog.ProvisioningInfoStateActive},
	}
	s.McpUserCredentials[service] = cred
	return Response{
		Body: cred,
	}
}

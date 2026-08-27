package testserver

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/databricks/databricks-sdk-go/service/catalog"
)

// ModelProviderServicesCreate fakes POST
// /api/2.1/unity-catalog/model-provider-services. See ModelServicesCreate for
// the shape: the ModelProviderService body is sent directly, parent and
// model_provider_service_id are query parameters, and the map is keyed by the
// {catalog}.{schema}.{model_provider_service} path segment.
func (s *FakeWorkspace) ModelProviderServicesCreate(req Request) Response {
	defer s.LockUnlock()()

	var ms catalog.ModelProviderService
	if err := json.Unmarshal(req.Body, &ms); err != nil {
		return Response{
			Body:       fmt.Sprintf("internal error: %s", err),
			StatusCode: http.StatusInternalServerError,
		}
	}

	schema := strings.TrimPrefix(req.URL.Query().Get("parent"), "schemas/")
	key := schema + "." + req.URL.Query().Get("model_provider_service_id")

	ms.Name = "model-provider-services/" + key
	ms.CreatedBy = s.CurrentUser().UserName
	ms.UpdatedBy = s.CurrentUser().UserName
	ms.EffectiveOwner = s.CurrentUser().UserName
	ms.MetastoreId = nextUUID()

	s.ModelProviderServices[key] = ms
	return Response{
		Body: ms,
	}
}

func (s *FakeWorkspace) ModelProviderServicesUpdate(req Request, name string) Response {
	defer s.LockUnlock()()

	existing, ok := s.ModelProviderServices[name]
	if !ok {
		return Response{
			StatusCode: http.StatusNotFound,
			Body:       fmt.Sprintf("model provider service %s not found", name),
		}
	}

	var incoming catalog.ModelProviderService
	if err := json.Unmarshal(req.Body, &incoming); err != nil {
		return Response{
			Body:       fmt.Sprintf("internal error: %s", err),
			StatusCode: http.StatusInternalServerError,
		}
	}

	existing.Comment = incoming.Comment
	existing.Config = incoming.Config
	existing.UpdatedBy = s.CurrentUser().UserName

	s.ModelProviderServices[name] = existing
	return Response{
		Body: existing,
	}
}

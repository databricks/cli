package testserver

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	sdktime "github.com/databricks/databricks-sdk-go/common/types/time"
	"github.com/databricks/databricks-sdk-go/service/catalog"
)

func init() {
	registerRoutes("model_provider_services", addModelProviderServicesRoutes)
	registerStore("model_provider_services", newModelProviderServicesStore)
}

type modelProviderServicesStore struct {
	ModelProviderServices map[string]catalog.ModelProviderService
}

func newModelProviderServicesStore(*FakeWorkspace) *modelProviderServicesStore {
	return &modelProviderServicesStore{
		ModelProviderServices: map[string]catalog.ModelProviderService{},
	}
}

func (s *FakeWorkspace) modelProviderServices() *modelProviderServicesStore {
	return stateOf[modelProviderServicesStore](s)
}

func addModelProviderServicesRoutes(server *Server) {
	// Model Provider Services (AI Gateway):

	server.Handle("POST", "/api/2.1/unity-catalog/model-provider-services", func(req Request) any {
		return req.Workspace.ModelProviderServicesCreate(req)
	})

	server.Handle("GET", "/api/2.1/unity-catalog/model-provider-services/{name}", func(req Request) any {
		return MapGet(req.Workspace, req.Workspace.modelProviderServices().ModelProviderServices, req.Vars["name"])
	})

	server.Handle("PATCH", "/api/2.1/unity-catalog/model-provider-services/{name}", func(req Request) any {
		return req.Workspace.ModelProviderServicesUpdate(req, req.Vars["name"])
	})

	server.Handle("DELETE", "/api/2.1/unity-catalog/model-provider-services/{name}", func(req Request) any {
		return MapDelete(req.Workspace, req.Workspace.modelProviderServices().ModelProviderServices, req.Vars["name"])
	})
}

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

	if _, exists := s.modelProviderServices().ModelProviderServices[key]; exists {
		return Response{
			StatusCode: http.StatusConflict,
			Body: map[string]string{
				"error_code": "ALREADY_EXISTS",
				"message":    fmt.Sprintf("ModelProviderService '%s' already exists", key),
			},
		}
	}

	ms.Name = "model-provider-services/" + key
	ms.CreatedBy = s.CurrentUser().UserName
	ms.UpdatedBy = s.CurrentUser().UserName
	ms.EffectiveOwner = s.CurrentUser().UserName
	ms.MetastoreId = nextUUID()
	ms.CreateTime = sdktime.New(time.Unix(0, nowNano()).UTC())

	s.modelProviderServices().ModelProviderServices[key] = ms
	return Response{
		Body: ms,
	}
}

func (s *FakeWorkspace) ModelProviderServicesUpdate(req Request, name string) Response {
	defer s.LockUnlock()()

	existing, ok := s.modelProviderServices().ModelProviderServices[name]
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

	s.modelProviderServices().ModelProviderServices[name] = existing
	return Response{
		Body: existing,
	}
}

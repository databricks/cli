package testserver

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/databricks/databricks-sdk-go/service/catalog"
)

// ModelServicesCreate fakes POST /api/2.1/unity-catalog/model-services.
//
// `parent` (schemas/{catalog}.{schema}) and `model_service_id` arrive as query
// parameters; the ModelService body is sent directly (not wrapped in
// "model_service"). The server derives the resource name
// model-services/{catalog}.{schema}.{model_service}.
// The map is keyed by the {catalog}.{schema}.{model_service} portion, which is
// the path segment used on subsequent get/update/delete.
func (s *FakeWorkspace) ModelServicesCreate(req Request) Response {
	defer s.LockUnlock()()

	// The SDK sends the ModelService body directly (not wrapped in
	// "model_service"); parent and model_service_id are query parameters.
	var ms catalog.ModelService
	if err := json.Unmarshal(req.Body, &ms); err != nil {
		return Response{
			Body:       fmt.Sprintf("internal error: %s", err),
			StatusCode: http.StatusInternalServerError,
		}
	}

	schema := strings.TrimPrefix(req.URL.Query().Get("parent"), "schemas/")
	key := schema + "." + req.URL.Query().Get("model_service_id")

	ms.Name = "model-services/" + key
	ms.CreatedBy = s.CurrentUser().UserName
	ms.UpdatedBy = s.CurrentUser().UserName
	ms.EffectiveOwner = s.CurrentUser().UserName
	ms.MetastoreId = nextUUID()

	s.ModelServices[key] = ms
	return Response{
		Body: ms,
	}
}

func (s *FakeWorkspace) ModelServicesUpdate(req Request, name string) Response {
	defer s.LockUnlock()()

	existing, ok := s.ModelServices[name]
	if !ok {
		return Response{
			StatusCode: http.StatusNotFound,
			Body:       fmt.Sprintf("model service %s not found", name),
		}
	}

	// The SDK sends the ModelService body directly; update_mask is a query param.
	var incoming catalog.ModelService
	if err := json.Unmarshal(req.Body, &incoming); err != nil {
		return Response{
			Body:       fmt.Sprintf("internal error: %s", err),
			StatusCode: http.StatusInternalServerError,
		}
	}

	// Apply the mutable fields carried in the update mask (comment, config).
	existing.Comment = incoming.Comment
	existing.Config = incoming.Config
	existing.UpdatedBy = s.CurrentUser().UserName

	s.ModelServices[name] = existing
	return Response{
		Body: existing,
	}
}

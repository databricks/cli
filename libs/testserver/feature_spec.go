package testserver

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/databricks/databricks-sdk-go/service/ml"
)

// FeatureSpecsCreate fakes POST /api/2.0/feature-engineering/feature-specs. The FeatureSpec body is
// sent directly, keyed by name, which is also the resource id. Like the real endpoint, it returns
// the feature spec it was sent, with the name lowercased.
func (s *FakeWorkspace) FeatureSpecsCreate(req Request) Response {
	defer s.LockUnlock()()

	var featureSpec ml.FeatureSpec
	if err := json.Unmarshal(req.Body, &featureSpec); err != nil {
		return Response{
			Body:       fmt.Sprintf("internal error: %s", err),
			StatusCode: http.StatusInternalServerError,
		}
	}

	// UC normalizes feature spec names to lowercase.
	featureSpec.Name = strings.ToLower(featureSpec.Name)

	if len(strings.Split(featureSpec.Name, ".")) != 3 {
		return Response{
			StatusCode: http.StatusBadRequest,
			Body:       fmt.Sprintf("name must be a three-part name, got %q", featureSpec.Name),
		}
	}
	if _, exists := s.FeatureSpecs[featureSpec.Name]; exists {
		return Response{
			StatusCode: http.StatusConflict,
			Body:       map[string]string{"error_code": "RESOURCE_ALREADY_EXISTS", "message": fmt.Sprintf("Function '%s' already exists", featureSpec.Name)},
		}
	}

	s.FeatureSpecs[featureSpec.Name] = featureSpec
	return Response{
		Body: featureSpec,
	}
}

// FeatureSpecsDelete fakes DELETE /api/2.0/feature-engineering/feature-specs/{name}. A feature spec
// is a UC function, and UC drops a function's grants with it, so a recreated feature spec starts
// with none.
func (s *FakeWorkspace) FeatureSpecsDelete(name string) Response {
	defer s.LockUnlock()()

	if _, ok := s.FeatureSpecs[name]; !ok {
		return Response{
			StatusCode: http.StatusNotFound,
			Body:       map[string]string{"error_code": "RESOURCE_DOES_NOT_EXIST", "message": fmt.Sprintf("Function '%s' does not exist.", name)},
		}
	}
	delete(s.FeatureSpecs, name)
	delete(s.Grants, grantsKey("function", name))
	return Response{Body: map[string]any{}}
}

package testserver

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/databricks/databricks-sdk-go/service/ml"
)

// FeaturesCreate fakes POST /api/2.0/feature-engineering/features. The Feature body is sent
// directly and the map is keyed by full_name, which is also the resource id. catalog_name,
// schema_name and name are derived here because the backend derives them from full_name.
func (s *FakeWorkspace) FeaturesCreate(req Request) Response {
	defer s.LockUnlock()()

	var feature ml.Feature
	if err := json.Unmarshal(req.Body, &feature); err != nil {
		return Response{
			Body:       fmt.Sprintf("internal error: %s", err),
			StatusCode: http.StatusInternalServerError,
		}
	}

	parts := strings.Split(feature.FullName, ".")
	if len(parts) != 3 {
		return Response{
			StatusCode: http.StatusBadRequest,
			Body:       fmt.Sprintf("full_name must be a three-part name, got %q", feature.FullName),
		}
	}
	feature.CatalogName, feature.SchemaName, feature.Name = parts[0], parts[1], parts[2]
	feature.CreatedBy = s.CurrentUser().UserName

	s.Features[feature.FullName] = feature
	return Response{
		Body: feature,
	}
}

// FeaturesUpdate fakes PATCH /api/2.0/feature-engineering/features/{full_name}. Only description
// is applied: it is the sole field the real update endpoint accepts.
func (s *FakeWorkspace) FeaturesUpdate(req Request, fullName string) Response {
	defer s.LockUnlock()()

	existing, ok := s.Features[fullName]
	if !ok {
		return Response{
			StatusCode: http.StatusNotFound,
			Body:       fmt.Sprintf("feature %s not found", fullName),
		}
	}

	var incoming ml.Feature
	if err := json.Unmarshal(req.Body, &incoming); err != nil {
		return Response{
			Body:       fmt.Sprintf("internal error: %s", err),
			StatusCode: http.StatusInternalServerError,
		}
	}

	existing.Description = incoming.Description

	s.Features[fullName] = existing
	return Response{
		Body: existing,
	}
}

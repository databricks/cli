package testserver

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	sdktime "github.com/databricks/databricks-sdk-go/common/types/time"
	"github.com/databricks/databricks-sdk-go/service/ml"
)

func init() {
	registerRoutes("feature", addFeatureRoutes)
	registerStore("feature", newFeaturesStore)
}

type featuresStore struct {
	Features map[string]ml.Feature
}

func newFeaturesStore(*FakeWorkspace) *featuresStore {
	return &featuresStore{
		Features: map[string]ml.Feature{},
	}
}

func (s *FakeWorkspace) features() *featuresStore {
	return stateOf[featuresStore](s)
}

func addFeatureRoutes(server *Server) {
	// Feature Engineering features:
	server.Handle("POST", "/api/2.0/feature-engineering/features", func(req Request) any {
		return req.Workspace.FeaturesCreate(req)
	})
	server.Handle("GET", "/api/2.0/feature-engineering/features/{full_name}", func(req Request) any {
		// Features are stored lowercased, and the cloud GET matches the name case-insensitively.
		return MapGet(req.Workspace, req.Workspace.features().Features, strings.ToLower(req.Vars["full_name"]))
	})
	server.Handle("PATCH", "/api/2.0/feature-engineering/features/{full_name}", func(req Request) any {
		return req.Workspace.FeaturesUpdate(req, req.Vars["full_name"])
	})
	server.Handle("DELETE", "/api/2.0/feature-engineering/features/{full_name}", func(req Request) any {
		return MapDelete(req.Workspace, req.Workspace.features().Features, req.Vars["full_name"])
	})
}

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

	// UC normalizes feature names to lowercase.
	feature.FullName = strings.ToLower(feature.FullName)

	parts := strings.Split(feature.FullName, ".")
	if len(parts) != 3 {
		return Response{
			StatusCode: http.StatusBadRequest,
			Body:       fmt.Sprintf("full_name must be a three-part name, got %q", feature.FullName),
		}
	}
	if _, exists := s.features().Features[feature.FullName]; exists {
		return Response{
			StatusCode: http.StatusConflict,
			Body: map[string]string{
				"error_code": "ALREADY_EXISTS",
				"message":    fmt.Sprintf("Feature '%s' already exists.", feature.FullName),
			},
		}
	}
	feature.CatalogName, feature.SchemaName, feature.Name = parts[0], parts[1], parts[2]
	feature.CreatedBy = s.CurrentUser().UserName
	feature.CreatedAt = sdktime.New(time.Unix(0, nowNano()).UTC())

	s.features().Features[feature.FullName] = feature
	return Response{
		Body: feature,
	}
}

// FeaturesUpdate fakes PATCH /api/2.0/feature-engineering/features/{full_name}. Only description
// is applied: it is the sole field the real update endpoint accepts.
func (s *FakeWorkspace) FeaturesUpdate(req Request, fullName string) Response {
	defer s.LockUnlock()()

	existing, ok := s.features().Features[fullName]
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

	s.features().Features[fullName] = existing
	return Response{
		Body: existing,
	}
}

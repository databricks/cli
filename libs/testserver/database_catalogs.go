package testserver

import (
	"encoding/json"
	"fmt"

	"github.com/databricks/databricks-sdk-go/service/database"
)

func (s *FakeWorkspace) DatabaseCatalogCreate(req Request) Response {
	defer s.LockUnlock()()

	databaseCatalog := database.DatabaseCatalog{}
	err := json.Unmarshal(req.Body, &databaseCatalog)
	if err != nil {
		return Response{
			Body:       fmt.Sprintf("cannot unmarshal request body: %v", err),
			StatusCode: 400,
		}
	}

	// check that the instance exists:
	found := false
	for _, instance := range s.DatabaseInstances {
		if instance.Name == databaseCatalog.DatabaseInstanceName {
			found = true
			break
		}
	}
	if !found {
		return Response{
			Body:       fmt.Sprintf("database instance with name '%s' not found", databaseCatalog.DatabaseInstanceName),
			StatusCode: 404,
		}
	}

	// Assign uid like the real backend does
	databaseCatalog.Uid = nextUUID()

	s.DatabaseCatalogs[databaseCatalog.Name] = databaseCatalog

	return Response{
		Body: databaseCatalog,
	}
}

func (s *FakeWorkspace) DatabaseCatalogGet(name string) Response {
	defer s.LockUnlock()()

	catalog, ok := s.DatabaseCatalogs[name]
	if !ok {
		return Response{
			Body:       fmt.Sprintf("database catalog with name '%s' not found", name),
			StatusCode: 404,
		}
	}

	// create_database_if_not_exists is a write-only field - the real API doesn't return it
	result := catalog
	result.CreateDatabaseIfNotExists = false

	return Response{Body: result}
}

// DatabaseCatalogUpdate models the real Database API, which has no update
// endpoint: PATCH /api/2.0/database/catalogs/{name} returns 501 NOT_IMPLEMENTED.
// The direct engine recreates database_catalogs on any change and never calls
// this; modeling the 501 guards against reintroducing an update path.
func (s *FakeWorkspace) DatabaseCatalogUpdate(req Request, name string) Response {
	return Response{
		StatusCode: 501,
		Body: map[string]string{
			"error_code": "NOT_IMPLEMENTED",
			"message":    "Update Database Catalog is not yet implemented",
		},
	}
}

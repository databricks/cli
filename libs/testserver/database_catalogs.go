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

	s.DatabaseCatalogs[databaseCatalog.Name] = databaseCatalog

	return Response{
		Body: databaseCatalog,
	}
}

func (s *FakeWorkspace) DatabaseCatalogUpdate(req Request, name string) Response {
	defer s.LockUnlock()()

	var updateRequest database.UpdateDatabaseCatalogRequest
	err := json.Unmarshal(req.Body, &updateRequest)
	if err != nil {
		return Response{
			Body:       fmt.Sprintf("cannot unmarshal request body: %v", err),
			StatusCode: 400,
		}
	}

	// Check if the catalog exists
	existingCatalog, exists := s.DatabaseCatalogs[name]
	if !exists {
		return Response{
			Body:       fmt.Sprintf("database catalog with name '%s' not found", name),
			StatusCode: 404,
		}
	}

	// Update the catalog with the new config
	updatedCatalog := updateRequest.DatabaseCatalog
	if updatedCatalog.Name == "" {
		updatedCatalog.Name = existingCatalog.Name
	}
	if updatedCatalog.DatabaseInstanceName == "" {
		updatedCatalog.DatabaseInstanceName = existingCatalog.DatabaseInstanceName
	}
	if updatedCatalog.DatabaseName == "" {
		updatedCatalog.DatabaseName = existingCatalog.DatabaseName
	}

	s.DatabaseCatalogs[name] = updatedCatalog

	return Response{
		Body: updatedCatalog,
	}
}

package testserver

import (
	"encoding/json"
	"fmt"

	"github.com/databricks/databricks-sdk-go/service/database"
)

func (w *FakeWorkspace) DatabaseCatalogCreate(req Request) Response {
	defer w.LockUnlock()()

	databaseCatalog := database.DatabaseCatalog{}
	err := json.Unmarshal(req.Body, &databaseCatalog)
	if err != nil {
		return Response{
			Body:       fmt.Sprintf("cannot unmarshal request body: %w", err),
			StatusCode: 400,
		}
	}

	// check that the instance exists:
	found := false
	for _, instance := range w.DatabaseInstances {
		if instance.Name == databaseCatalog.DatabaseInstanceName {
			fmt.Printf("Found database instance: %s\n", instance.Name)
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

	w.DatabaseCatalogs[databaseCatalog.Name] = databaseCatalog

	return Response{
		Body: databaseCatalog,
	}
}

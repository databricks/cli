package testserver

import (
	"encoding/json"
	"fmt"

	"github.com/databricks/databricks-sdk-go/service/database"
	"github.com/google/uuid"
)

func (s *FakeWorkspace) DatabaseCatalogCreate(req Request) Response {
	defer s.LockUnlock()()

	databaseCatalog := database.DatabaseCatalog{}
	err := json.Unmarshal(req.Body, &databaseCatalog)
	if err != nil {
		return Response{
			Body:       fmt.Sprintf("cannot unmarshal request body: %s", err),
			StatusCode: 400,
		}
	}

	databaseCatalog.Uid = uuid.New().String()
	s.DatabaseCatalogs[databaseCatalog.Name] = databaseCatalog

	return Response{
		Body: databaseCatalog,
	}
}

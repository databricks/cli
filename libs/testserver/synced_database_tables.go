package testserver

import (
	"encoding/json"
	"fmt"

	"github.com/databricks/databricks-sdk-go/service/database"
)

func (s *FakeWorkspace) SyncedDatabaseTableCreate(req Request) Response {
	defer s.LockUnlock()()

	syncedDatabaseTable := database.SyncedDatabaseTable{}
	err := json.Unmarshal(req.Body, &syncedDatabaseTable)
	if err != nil {
		return Response{
			Body:       fmt.Sprintf("cannot unmarshal request body: %v", err),
			StatusCode: 400,
		}
	}

	// check that the database instance exists if specified:
	if syncedDatabaseTable.DatabaseInstanceName != "" {
		found := false
		for _, instance := range s.DatabaseInstances {
			if instance.Name == syncedDatabaseTable.DatabaseInstanceName {
				found = true
				break
			}
		}
		if !found {
			return Response{
				Body:       fmt.Sprintf("database instance with name '%s' not found", syncedDatabaseTable.DatabaseInstanceName),
				StatusCode: 404,
			}
		}
	}

	s.SyncedDatabaseTables[syncedDatabaseTable.Name] = syncedDatabaseTable

	return Response{
		Body: syncedDatabaseTable,
	}
}

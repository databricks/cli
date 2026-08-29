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

// SyncedDatabaseTableUpdate models the real Database API, which has no update
// endpoint: PATCH /api/2.0/database/synced_tables/{name} returns 501
// NOT_IMPLEMENTED. The direct engine recreates synced_database_tables on any
// change and never calls this; modeling the 501 guards against reintroducing an
// update path.
func (s *FakeWorkspace) SyncedDatabaseTableUpdate(req Request, name string) Response {
	return Response{
		StatusCode: 501,
		Body: map[string]string{
			"error_code": "NOT_IMPLEMENTED",
			"message":    "Update Synced Database Table is not yet implemented.",
		},
	}
}

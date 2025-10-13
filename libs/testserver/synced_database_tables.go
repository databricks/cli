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

func (s *FakeWorkspace) SyncedDatabaseTableUpdate(req Request, name string) Response {
	defer s.LockUnlock()()

	var updateReq database.UpdateSyncedDatabaseTableRequest
	if err := json.Unmarshal(req.Body, &updateReq); err != nil {
		return Response{
			Body:       fmt.Sprintf("cannot unmarshal request body: %v", err),
			StatusCode: 400,
		}
	}

	// Ensure the resource exists
	existing, ok := s.SyncedDatabaseTables[name]
	if !ok {
		return Response{
			Body:       fmt.Sprintf("synced database table with name '%s' not found", name),
			StatusCode: 404,
		}
	}

	// Apply updates: shallow replace with provided struct while preserving name
	updated := updateReq.SyncedTable
	if updated.Name == "" {
		updated.Name = existing.Name
	}

	s.SyncedDatabaseTables[name] = updated

	return Response{Body: updated}
}

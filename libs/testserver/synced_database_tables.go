package testserver

import (
	"encoding/json"
	"fmt"

	"github.com/databricks/databricks-sdk-go/service/database"
)

func init() {
	registerRoutes("synced_database_tables", addSyncedDatabaseTablesRoutes)
	registerStore("synced_database_tables", newSyncedDatabaseTablesStore)
}

type syncedDatabaseTablesStore struct {
	SyncedDatabaseTables map[string]database.SyncedDatabaseTable
}

func newSyncedDatabaseTablesStore(*FakeWorkspace) *syncedDatabaseTablesStore {
	return &syncedDatabaseTablesStore{
		SyncedDatabaseTables: map[string]database.SyncedDatabaseTable{},
	}
}

func (s *FakeWorkspace) syncedDatabaseTables() *syncedDatabaseTablesStore {
	return stateOf[syncedDatabaseTablesStore](s)
}

func addSyncedDatabaseTablesRoutes(server *Server) {
	// Synced Database Tables:

	server.Handle("POST", "/api/2.0/database/synced_tables", func(req Request) any {
		return req.Workspace.SyncedDatabaseTableCreate(req)
	})

	server.Handle("PATCH", "/api/2.0/database/synced_tables/{name}", func(req Request) any {
		return req.Workspace.SyncedDatabaseTableUpdate(req, req.Vars["name"])
	})

	server.Handle("GET", "/api/2.0/database/synced_tables/{name}", func(req Request) any {
		return MapGet(req.Workspace, req.Workspace.syncedDatabaseTables().SyncedDatabaseTables, req.Vars["name"])
	})

	server.Handle("DELETE", "/api/2.0/database/synced_tables/{name}", func(req Request) any {
		return MapDelete(req.Workspace, req.Workspace.syncedDatabaseTables().SyncedDatabaseTables, req.Vars["name"])
	})
}

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
		for _, instance := range s.databaseInstances().DatabaseInstances {
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

	s.syncedDatabaseTables().SyncedDatabaseTables[syncedDatabaseTable.Name] = syncedDatabaseTable

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

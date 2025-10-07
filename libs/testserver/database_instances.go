package testserver

import (
	"encoding/json"
	"fmt"
	"slices"
	"time"

	"github.com/databricks/databricks-sdk-go/service/database"
	"github.com/google/uuid"
)

var ForceSendFields []string = []string{
	"EffectiveEnableReadableSecondaries",
	"EffectiveStopped",
	"EffectiveEnablePgNativeLogin",
}

func (s *FakeWorkspace) DatabaseInstanceCreate(req Request) Response {
	defer s.LockUnlock()()

	databaseInstance := database.DatabaseInstance{}
	err := json.Unmarshal(req.Body, &databaseInstance)
	if err != nil {
		return Response{
			Body:       fmt.Sprintf("cannot unmarshal request body: %s", err),
			StatusCode: 400,
		}
	}

	// set default fields:
	databaseInstance.Uid = uuid.New().String()
	databaseInstance.State = database.DatabaseInstanceStateAvailable
	databaseInstance.PgVersion = "PG_VERSION_16"
	databaseInstance.EffectiveNodeCount = 1
	databaseInstance.EffectiveRetentionWindowInDays = 7
	databaseInstance.EffectiveCapacity = databaseInstance.Capacity
	databaseInstance.Creator = req.Workspace.CurrentUser().UserName
	databaseInstance.CreationTime = time.Now().UTC().Format(time.RFC3339)
	databaseInstance.EffectiveEnableReadableSecondaries = false
	databaseInstance.EffectiveStopped = false
	databaseInstance.EffectiveEnablePgNativeLogin = false

	databaseInstance.ForceSendFields = slices.Clone(ForceSendFields)

	s.DatabaseInstances[databaseInstance.Name] = databaseInstance

	return Response{
		Body: databaseInstance,
	}
}

// DatabaseInstanceMapGet ensures false boolean values are returned
func DatabaseInstanceMapGet(w *FakeWorkspace, collection map[string]database.DatabaseInstance, key string) Response {
	defer w.LockUnlock()()

	value, ok := collection[key]
	if !ok {
		return Response{
			StatusCode: 404,
			Body:       map[string]string{"message": fmt.Sprintf("DatabaseInstance not found: %v", key)},
		}
	}

	value.ForceSendFields = slices.Clone(ForceSendFields)

	return Response{
		Body: value,
	}
}

func (s *FakeWorkspace) DatabaseInstanceUpdate(req Request, name string) Response {
	defer s.LockUnlock()()

	// Parse the update request
	var updateReq database.UpdateDatabaseInstanceRequest
	err := json.Unmarshal(req.Body, &updateReq)
	if err != nil {
		return Response{
			Body:       fmt.Sprintf("cannot unmarshal request body: %s", err),
			StatusCode: 400,
		}
	}

	// Check if the instance exists
	existing, ok := s.DatabaseInstances[name]
	if !ok {
		return Response{
			StatusCode: 404,
			Body:       map[string]string{"message": fmt.Sprintf("DatabaseInstance not found: %v", name)},
		}
	}

	// Update the instance with new values while preserving system-managed fields
	updated := updateReq.DatabaseInstance
	updated.Uid = existing.Uid                   // Preserve UID
	updated.Creator = existing.Creator           // Preserve creator
	updated.CreationTime = existing.CreationTime // Preserve creation time
	updated.State = existing.State               // Preserve state

	// Set defaults for effective fields if not specified
	if updated.EffectiveNodeCount == 0 {
		updated.EffectiveNodeCount = existing.EffectiveNodeCount
	}
	if updated.EffectiveRetentionWindowInDays == 0 {
		updated.EffectiveRetentionWindowInDays = existing.EffectiveRetentionWindowInDays
	}
	if updated.PgVersion == "" {
		updated.PgVersion = existing.PgVersion
	}

	// Set ForceSendFields to ensure consistent serialization with DatabaseInstanceMapGet
	updated.ForceSendFields = slices.Clone(ForceSendFields)

	// Update the stored instance
	s.DatabaseInstances[name] = updated

	return Response{
		Body: updated,
	}
}

func DatabaseInstanceMapDelete(req Request) Response {
	// Check if purge parameter is set to true in query parameters
	purge := req.URL.Query().Get("purge") == "true"
	if !purge {
		return Response{
			Body:       map[string]string{"message": "DELETE request must have purge=true parameter"},
			StatusCode: 400,
		}
	}

	return MapDelete(req.Workspace, req.Workspace.DatabaseInstances, req.Vars["name"])
}

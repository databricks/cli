package testserver

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/databricks/databricks-sdk-go/service/database"
	"github.com/google/uuid"
)

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
	databaseInstance.Creator = req.Workspace.CurrentUser().UserName
	databaseInstance.CreationTime = time.Now().UTC().Format(time.RFC3339)
	databaseInstance.EffectiveEnableReadableSecondaries = false
	databaseInstance.EffectiveStopped = false

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

	// Convert to map[string]interface{} to ensure all fields are included
	jsonBytes, _ := json.Marshal(value)
	var result map[string]any
	_ = json.Unmarshal(jsonBytes, &result)

	// Explicitly set boolean fields that should always be present
	result["effective_enable_readable_secondaries"] = value.EffectiveEnableReadableSecondaries
	result["effective_stopped"] = value.EffectiveStopped

	return Response{
		Body: result,
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

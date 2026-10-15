package testserver

import (
	"encoding/json"
	"fmt"
	"time"

	database "github.com/databricks/cli/libs/sdktypes/database/v1"
	"github.com/databricks/cli/libs/utils"
	"github.com/databricks/sdk-go/core/types"
)

// setEffectiveBools makes the effective_* booleans always serialize, false
// included, as the real API returns them.
func setEffectiveBools(d *database.DatabaseInstance) {
	d.EffectiveEnableReadableSecondaries = new(utils.Deref(d.EffectiveEnableReadableSecondaries))
	d.EffectiveStopped = new(utils.Deref(d.EffectiveStopped))
	d.EffectiveEnablePgNativeLogin = new(utils.Deref(d.EffectiveEnablePgNativeLogin))
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
	databaseInstance.Uid = utils.NonZero(nextUUID())
	databaseInstance.State = database.DatabaseInstance_State_Available
	databaseInstance.PgVersion = utils.NonZero("PG_VERSION_16")
	databaseInstance.EffectiveNodeCount = utils.NonZero(1)
	databaseInstance.EffectiveRetentionWindowInDays = utils.NonZero(7)
	databaseInstance.EffectiveCapacity = databaseInstance.Capacity
	databaseInstance.Creator = req.Workspace.CurrentUser().UserName
	databaseInstance.CreationTime = types.NewFromTime(time.Now().UTC().Truncate(time.Second))
	databaseInstance.EffectiveEnableReadableSecondaries = nil
	databaseInstance.EffectiveStopped = nil
	databaseInstance.EffectiveEnablePgNativeLogin = nil
	setEffectiveBools(&databaseInstance)

	s.DatabaseInstances[utils.Deref(databaseInstance.Name)] = databaseInstance

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

	setEffectiveBools(&value)

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
	updated := utils.Deref(updateReq.DatabaseInstance)
	updated.Uid = existing.Uid                   // Preserve UID
	updated.Creator = existing.Creator           // Preserve creator
	updated.CreationTime = existing.CreationTime // Preserve creation time
	updated.State = existing.State               // Preserve state

	// Set defaults for effective fields if not specified
	if utils.Deref(updated.EffectiveNodeCount) == 0 {
		updated.EffectiveNodeCount = existing.EffectiveNodeCount
	}
	if utils.Deref(updated.EffectiveRetentionWindowInDays) == 0 {
		updated.EffectiveRetentionWindowInDays = existing.EffectiveRetentionWindowInDays
	}
	if utils.Deref(updated.PgVersion) == "" {
		updated.PgVersion = existing.PgVersion
	}

	// Ensure consistent serialization with DatabaseInstanceMapGet
	setEffectiveBools(&updated)

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

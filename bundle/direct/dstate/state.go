package dstate

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sync"

	"github.com/databricks/cli/bundle/statemgmt/resourcestate"
	"github.com/google/uuid"
)

type DeploymentState struct {
	Path string
	Data Database
	mu   sync.Mutex
}

type Database struct {
	Lineage         string                              `json:"lineage"`
	Serial          int                                 `json:"serial"`
	DeploymentUnits map[string]map[string]ResourceEntry `json:"resources"`
}

type ResourceEntry struct {
	ID    string `json:"__id__"`
	State any    `json:"state"`
}

func (db *DeploymentState) SaveState(group, resourceName, newID string, state any) error {
	db.AssertOpened()
	db.mu.Lock()
	defer db.mu.Unlock()

	groupData, ok := db.Data.DeploymentUnits[group]
	if !ok {
		groupData = make(map[string]ResourceEntry)
		db.Data.DeploymentUnits[group] = groupData
	}

	groupData[resourceName] = ResourceEntry{
		ID:    newID,
		State: state,
	}

	return nil
}

func (db *DeploymentState) DeleteState(group, resourceName string) error {
	db.AssertOpened()
	db.mu.Lock()
	defer db.mu.Unlock()

	groupData, ok := db.Data.DeploymentUnits[group]
	if !ok {
		return nil
	}

	delete(groupData, resourceName)
	if len(groupData) == 0 {
		delete(db.Data.DeploymentUnits, group)
	}

	return nil
}

func (db *DeploymentState) GetResourceEntry(group, resourceName string) (ResourceEntry, bool) {
	db.AssertOpened()
	db.mu.Lock()
	defer db.mu.Unlock()

	groupData, ok := db.Data.DeploymentUnits[group]
	if !ok {
		return ResourceEntry{}, false
	}

	result, ok := groupData[resourceName]
	return result, ok
}

func (db *DeploymentState) Open(path string) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	if db.Path != "" {
		if db.Path == path {
			return nil
		}
		return fmt.Errorf("already read state %v, cannot open %v", db.Path, path)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			db.Data = Database{
				Serial:          0,
				Lineage:         uuid.New().String(),
				DeploymentUnits: make(map[string]map[string]ResourceEntry),
			}
			db.Path = path
			return nil
		}
		return err
	}

	err = json.Unmarshal(data, &db.Data)
	if err != nil {
		return err
	}

	db.Path = path
	return nil
}

func (db *DeploymentState) Finalize() error {
	db.mu.Lock()
	defer db.mu.Unlock()

	return db.unlockedSave()
}

func (db *DeploymentState) AssertOpened() {
	if db.Path == "" {
		panic("internal error: DeploymentState must be opened first")
	}
}

func (db *DeploymentState) ExportState(ctx context.Context) resourcestate.ExportedResourcesMap {
	result := make(resourcestate.ExportedResourcesMap, len(db.Data.DeploymentUnits))
	for groupName, group := range db.Data.DeploymentUnits {
		resultGroup := make(map[string]resourcestate.ResourceState, len(group))
		result[groupName] = resultGroup
		for resourceName, entry := range group {
			resultGroup[resourceName] = resourcestate.ResourceState{
				ID: entry.ID,
				// TODO: extract Etag
			}
		}
	}
	return result
}

func (db *DeploymentState) unlockedSave() error {
	db.AssertOpened()
	data, err := json.MarshalIndent(db.Data, "", " ")
	if err != nil {
		return err
	}

	err = os.WriteFile(db.Path, data, 0o600)
	if err != nil {
		return fmt.Errorf("failed to save resources state to %#v: %w", db.Path, err)
	}

	return nil
}

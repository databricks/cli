package dstate

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
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
	Lineage string                   `json:"lineage"`
	Serial  int                      `json:"serial"`
	State   map[string]ResourceEntry `json:"state"`
}

type ResourceEntry struct {
	ID    string `json:"__id__"`
	State any    `json:"state"`
}

// splitKey parses a canonical key back into group and name.
func splitKey(key string) (group, name string, ok bool) {
	const prefix = "resources."
	if !strings.HasPrefix(key, prefix) {
		return "", "", false
	}
	rest := strings.TrimPrefix(key, prefix)
	idx := strings.IndexByte(rest, '.')
	if idx <= 0 || idx >= len(rest)-1 {
		return "", "", false
	}
	return rest[:idx], rest[idx+1:], true
}

func (db *DeploymentState) SaveState(key, newID string, state any) error {
	db.AssertOpened()
	db.mu.Lock()
	defer db.mu.Unlock()

	if db.Data.State == nil {
		db.Data.State = make(map[string]ResourceEntry)
	}

	db.Data.State[key] = ResourceEntry{
		ID:    newID,
		State: state,
	}

	return nil
}

func (db *DeploymentState) DeleteState(key string) error {
	db.AssertOpened()
	db.mu.Lock()
	defer db.mu.Unlock()

	if db.Data.State == nil {
		return nil
	}

	delete(db.Data.State, key)

	return nil
}

func (db *DeploymentState) GetResourceEntry(key string) (ResourceEntry, bool) {
	db.AssertOpened()
	db.mu.Lock()
	defer db.mu.Unlock()

	if db.Data.State == nil {
		return ResourceEntry{}, false
	}

	result, ok := db.Data.State[key]
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
				Serial:  0,
				Lineage: uuid.New().String(),
				State:   make(map[string]ResourceEntry),
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
	result := make(resourcestate.ExportedResourcesMap)
	for key, entry := range db.Data.State {
		groupName, resourceName, ok := splitKey(key)
		if !ok {
			continue
		}
		resultGroup, ok := result[groupName]
		if !ok {
			resultGroup = make(map[string]resourcestate.ResourceState)
			result[groupName] = resultGroup
		}
		// Extract etag for dashboards.
		var etag string
		if dashboard, ok := entry.State.(map[string]any); groupName == "dashboards" && ok {
			v, ok := dashboard["etag"].(string)
			if ok {
				etag = v
			}
		}

		resultGroup[resourceName] = resourcestate.ResourceState{
			ID:   entry.ID,
			ETag: etag,
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

package dstate

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync"

	"github.com/databricks/cli/bundle/config/resources"
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

// splitKey extracts group and name from the key: 'resources.jobs.foo' -> 'jobs', 'foo', true
// For sub-resources like permissions it returns "", "", false
// Note we don't use group/name anywhere in bundle/direct, this is only for ExportState
// which makes ID available to other parts of DABs
func splitKey(key string) (group, name string, ok bool) {
	items := strings.Split(key, ".")
	if len(items) != 3 {
		// e.g. resources.jobs.foo.permissions
		return "", "", false
	}
	if items[0] != "resources" {
		return "", "", false
	}
	return items[1], items[2], true
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
		switch dashboard := entry.State.(type) {
		// Dashboard state has type map[string]any during bundle deployment.
		// covered by test case: bundle/deploy/dashboard/detect-change
		case map[string]any:
			v, ok := dashboard["etag"].(string)
			if ok {
				etag = v
			}

		// Dashboard state has type *resources.DashboardConfig during bundle generation.
		// covered by test case: bundle/deploy/dashboard/generate_inplace
		case *resources.DashboardConfig:
			etag = dashboard.Etag
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

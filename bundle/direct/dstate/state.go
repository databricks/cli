package dstate

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
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
	WAL     []LogEntry               `json:"wal,omitempty"`
}

type LogEntry struct {
	Op    string `json:"op"`
	Key   string `json:"key"`
	ID    string `json:"id,omitempty"`
	State any    `json:"state,omitempty"`
}

type ResourceEntry struct {
	ID    string `json:"__id__"`
	State any    `json:"state"`
}

func (db *DeploymentState) SaveState(key, newID string, state any) error {
	db.AssertOpened()
	db.mu.Lock()
	defer db.mu.Unlock()

	db.Data.WAL = append(db.Data.WAL, LogEntry{Op: "save", Key: key, ID: newID, State: state})
	if err := db.writeState(); err != nil {
		db.Data.WAL = db.Data.WAL[:len(db.Data.WAL)-1] // Rollback on failure
		return err
	}

	if db.Data.State == nil {
		db.Data.State = make(map[string]ResourceEntry)
	}
	db.Data.State[key] = ResourceEntry{ID: newID, State: state}
	return nil
}

func (db *DeploymentState) DeleteState(key string) error {
	db.AssertOpened()
	db.mu.Lock()
	defer db.mu.Unlock()

	db.Data.WAL = append(db.Data.WAL, LogEntry{Op: "delete", Key: key})
	if err := db.writeState(); err != nil {
		db.Data.WAL = db.Data.WAL[:len(db.Data.WAL)-1] // Rollback on failure
		return err
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
				Serial:  1,
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
	db.replayWAL()
	return nil
}

func (db *DeploymentState) Finalize() error {
	db.mu.Lock()
	defer db.mu.Unlock()

	oldWAL := db.Data.WAL
	db.Data.WAL = nil
	if err := db.writeState(); err != nil {
		db.Data.WAL = oldWAL // Restore on failure
		return err
	}
	return nil
}

func (db *DeploymentState) AssertOpened() {
	if db.Path == "" {
		panic("internal error: DeploymentState must be opened first")
	}
}

func (db *DeploymentState) ExportState(ctx context.Context) resourcestate.ExportedResourcesMap {
	result := make(resourcestate.ExportedResourcesMap)
	for key, entry := range db.Data.State {
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

		result[key] = resourcestate.ResourceState{
			ID:   entry.ID,
			ETag: etag,
		}
	}
	return result
}

func (db *DeploymentState) writeState() error {
	db.AssertOpened()
	data, err := json.MarshalIndent(db.Data, "", " ")
	if err != nil {
		return err
	}
	return os.WriteFile(db.Path, data, 0o600)
}

func (db *DeploymentState) replayWAL() {
	for _, entry := range db.Data.WAL {
		switch entry.Op {
		case "save":
			if db.Data.State == nil {
				db.Data.State = make(map[string]ResourceEntry)
			}
			db.Data.State[entry.Key] = ResourceEntry{ID: entry.ID, State: entry.State}
		case "delete":
			delete(db.Data.State, entry.Key)
		}
	}
}

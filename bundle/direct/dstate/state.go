package dstate

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync"

	"github.com/databricks/cli/bundle/deployplan"
	"github.com/databricks/cli/bundle/statemgmt/resourcestate"
	"github.com/databricks/cli/internal/build"
	"github.com/google/uuid"
)

const currentStateVersion = 1

type DeploymentState struct {
	Path string
	Data Database
	mu   sync.Mutex
}

type Database struct {
	StateVersion int                      `json:"state_version"`
	CLIVersion   string                   `json:"cli_version"`
	Lineage      string                   `json:"lineage"`
	Serial       int                      `json:"serial"`
	State        map[string]ResourceEntry `json:"state"`
}

type ResourceEntry struct {
	ID        string                      `json:"__id__"`
	State     json.RawMessage             `json:"state"`
	DependsOn []deployplan.DependsOnEntry `json:"depends_on,omitempty"`
}

func NewDatabase() Database {
	return NewMigratedDatabase(uuid.New().String(), 1)
}

func NewMigratedDatabase(lineage string, serial int) Database {
	return Database{
		StateVersion: currentStateVersion,
		CLIVersion:   build.GetInfo().Version,
		Lineage:      lineage,
		Serial:       serial,
		State:        make(map[string]ResourceEntry),
	}
}

func (db *DeploymentState) SaveState(key, newID string, state any, dependsOn []deployplan.DependsOnEntry) error {
	db.AssertOpened()
	db.mu.Lock()
	defer db.mu.Unlock()

	if db.Data.State == nil {
		db.Data.State = make(map[string]ResourceEntry)
	}

	jsonMessage, err := json.MarshalIndent(state, "  ", " ")
	if err != nil {
		return err
	}

	db.Data.State[key] = ResourceEntry{
		ID:        newID,
		State:     json.RawMessage(jsonMessage),
		DependsOn: dependsOn,
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
			db.Data = NewDatabase()
			db.Path = path
			return nil
		}
		return err
	}
	// TODO: Use json.Decoder with UseNumber() to preserve precision of large integers.
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

	db.Data.Serial += 1

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
		var etag string
		// Extract etag for dashboards.
		// covered by test case: bundle/deploy/dashboard/detect-change
		if strings.Contains(key, ".dashboards.") && len(entry.State) > 0 {
			var holder struct {
				Etag string `json:"etag"`
			}
			if err := json.Unmarshal(entry.State, &holder); err == nil {
				etag = holder.Etag
			}
		}

		result[key] = resourcestate.ResourceState{
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

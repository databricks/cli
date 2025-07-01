package tnstate

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"

	"github.com/google/uuid"
)

type TerranovaState struct {
	Path string
	Data Database
	mu   sync.Mutex
}

// Loading resources.json with state_version higher than CurrentStateVersion is never supported
// Loading resources.json with state_version lower than CurrentStateVersion may be supported

const CurrentStateVersion = 0

type Database struct {
	Lineage      string                              `json:"lineage"`
	Serial       int                                 `json:"serial"`
	StateVersion int                                 `json:"state_version"`
	CLIVersion   CLIVersion                          `json:"cli_version"`
	Resources    map[string]map[string]ResourceEntry `json:"resources"`
}

type ResourceEntry struct {
	ID    string `json:"__id__"`
	State any    `json:"state"`
}

func (db *TerranovaState) SaveState(group, resourceName, newID string, state any) error {
	db.AssertOpened()
	db.mu.Lock()
	defer db.mu.Unlock()

	groupData, ok := db.Data.Resources[group]
	if !ok {
		groupData = make(map[string]ResourceEntry)
		db.Data.Resources[group] = groupData
	}

	groupData[resourceName] = ResourceEntry{
		ID:    newID,
		State: state,
	}

	return nil
}

func (db *TerranovaState) DeleteState(group, resourceName string) error {
	db.AssertOpened()
	db.mu.Lock()
	defer db.mu.Unlock()

	groupData, ok := db.Data.Resources[group]
	if !ok {
		return nil
	}

	delete(groupData, resourceName)

	return nil
}

func (db *TerranovaState) GetResourceEntry(group, resourceName string) (ResourceEntry, bool) {
	db.AssertOpened()
	db.mu.Lock()
	defer db.mu.Unlock()

	groupData, ok := db.Data.Resources[group]
	if !ok {
		return ResourceEntry{}, false
	}

	result, ok := groupData[resourceName]
	return result, ok
}

func (db *TerranovaState) Open(path string, cliVersion CLIVersion) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	db.Path = path

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			db.Data = Database{
				Serial:       0,
				Lineage:      uuid.New().String(),
				StateVersion: CurrentStateVersion,
				CLIVersion:   cliVersion,
				Resources:    make(map[string]map[string]ResourceEntry),
			}
			return nil
		}
		return err
	}

	err = json.Unmarshal(data, &db.Data)
	if err != nil {
		return err
	}

	if db.Data.StateVersion > CurrentStateVersion {
		return fmt.Errorf("state file format has version=%d (produced by CLI %s) which is higher than version=%d than your CLI %s supports. Upgrade of the CLI is required", db.Data.StateVersion, db.Data.CLIVersion, CurrentStateVersion, cliVersion)
	}

	return nil
}

func (db *TerranovaState) Finalize() error {
	db.mu.Lock()
	defer db.mu.Unlock()

	return db.unlockedSave()
}

func (db *TerranovaState) AssertOpened() {
	if db.Path == "" {
		panic("internal error: TerranovaState must be opened first")
	}
}

func (db *TerranovaState) unlockedSave() error {
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

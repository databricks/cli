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

type Database struct {
	Lineage   string                              `json:"lineage"`
	Serial    int                                 `json:"serial"`
	Resources map[string]map[string]ResourceEntry `json:"resources"`
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

func (db *TerranovaState) Open(path string) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	db.Path = path

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			db.Data = Database{
				Serial:    0,
				Lineage:   uuid.New().String(),
				Resources: make(map[string]map[string]ResourceEntry),
			}
			return nil
		}
		return err
	}

	return json.Unmarshal(data, &db.Data)
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

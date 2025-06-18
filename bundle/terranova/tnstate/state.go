package tnstate

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"

	"github.com/databricks/cli/libs/utils"
	"github.com/google/uuid"
)

type TerranovaState struct {
	Path string
	data Database
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

type ResourceNode struct {
	Group string
	Name  string
	ID    string
}

func (db *TerranovaState) SaveState(section, resourceName, newID string, state any) error {
	db.assertOpened()
	db.mu.Lock()
	defer db.mu.Unlock()

	sectionData, ok := db.data.Resources[section]
	if !ok {
		sectionData = make(map[string]ResourceEntry)
		db.data.Resources[section] = sectionData
	}

	sectionData[resourceName] = ResourceEntry{
		ID:    newID,
		State: state,
	}

	return nil
}

func (db *TerranovaState) DeleteState(section, resourceName string) error {
	db.assertOpened()
	db.mu.Lock()
	defer db.mu.Unlock()

	sectionData, ok := db.data.Resources[section]
	if !ok {
		return nil
	}

	delete(sectionData, resourceName)

	return nil
}

func (db *TerranovaState) GetResourceEntry(section, resourceName string) (ResourceEntry, bool) {
	db.assertOpened()
	db.mu.Lock()
	defer db.mu.Unlock()

	sectionData, ok := db.data.Resources[section]
	if !ok {
		return ResourceEntry{}, false
	}

	result, ok := sectionData[resourceName]
	return result, ok
}

/*
func (db *TerranovaState) GetAllResourcesMap() map[string]map[string]any {
	db.assertOpened()

	nodes := make(map[string]map[string]any)

	for _, group := range utils.SortedKeys(db.data.Resources) {
		submap, ok := nodes[group]
		if !ok {
			submap = make(map[string]any)
			nodes[group] = submap
		}

		sectionData := db.data.Resources[group]
		for _, name := range sectionData {
			submap[name] = nil
		}
	}

	return nodes
}*/

func (db *TerranovaState) GetAllResources() []ResourceNode {
	db.assertOpened()

	nodes := make([]ResourceNode, 0, len(db.data.Resources)*4)

	for _, section := range utils.SortedKeys(db.data.Resources) {
		sectionData := db.data.Resources[section]
		for _, name := range utils.SortedKeys(sectionData) {
			nodes = append(nodes, ResourceNode{
				Group: section,
				Name:  name,
				ID:    sectionData[name].ID,
			})
		}
	}

	return nodes
}

func (db *TerranovaState) Open(path string) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	db.Path = path

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			db.data = Database{
				Serial:    0,
				Lineage:   uuid.New().String(),
				Resources: make(map[string]map[string]ResourceEntry),
			}
			return nil
		}
		return err
	}

	return json.Unmarshal(data, &db.data)
}

func (db *TerranovaState) Finalize() error {
	db.mu.Lock()
	defer db.mu.Unlock()

	return db.unlockedSave()
}

func (db *TerranovaState) assertOpened() {
	if db.Path == "" {
		panic("internal error: TerranovaState must be opened first")
	}
}

func (db *TerranovaState) unlockedSave() error {
	db.assertOpened()
	data, err := json.MarshalIndent(db.data, "", " ")
	if err != nil {
		return err
	}

	err = os.WriteFile(db.Path, data, 0o600)
	if err != nil {
		return fmt.Errorf("failed to save resources state to %#v: %w", db.Path, err)
	}

	return nil
}

func (r ResourceNode) String() string {
	return r.Group + "." + r.Name + "#" + r.ID
}

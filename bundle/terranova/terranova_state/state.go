package terranova_state

import (
	"bytes"
	"encoding/json"
	"os"
	"reflect"
	"sync"

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

func (db *TerranovaState) SaveState(section, resourceName, newID string, state any) error {
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

func jsonRoundTrip(src, dest any) error {
	raw, err := json.Marshal(src)
	if err != nil {
		return err
	}

	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.UseNumber()
	dec.DisallowUnknownFields()

	return dec.Decode(dest)
}

func (db *TerranovaState) GetSavedState(section, resourceName string, stateType reflect.Type) (any, error) {
	sectionData, ok := db.data.Resources[section]
	if !ok {
		return nil, nil
	}

	entry, ok := sectionData[resourceName]
	if !ok {
		return nil, nil
	}

	destPtr := reflect.New(stateType).Interface()
	err := jsonRoundTrip(entry.State, destPtr)
	if err != nil {
		return nil, err
	}

	return reflect.ValueOf(destPtr).Elem().Interface(), nil
}

func (db *TerranovaState) GetResourceID(section, resourceName string) (string, error) {
	db.mu.Lock()
	defer db.mu.Unlock()

	sectionData, ok := db.data.Resources[section]
	if !ok {
		return "", nil
	}

	return sectionData[resourceName].ID, nil
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
			return db.unlockedSave()
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

func (db *TerranovaState) unlockedSave() error {
	data, err := json.MarshalIndent(db.data, "", " ")
	if err != nil {
		return err
	}

	return os.WriteFile(db.Path, data, 0o600)
}

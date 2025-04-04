package db

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type ResourceRow struct {
	// PK: Section, Name
	Config     string
	ResourceID string
	PreTS      int64
	PostTS     int64
}

type ResourcesMap map[string]*ResourceRow

type SimpleDB struct {
	Resources ResourcesMap `json:"resources"`
}

type JSONFileDB struct {
	Path string
	DB   SimpleDB
	mu   sync.Mutex
}

func (db ResourcesMap) GetRow(ctx context.Context, section, resourceName string) (ResourceRow, bool) {
	key := section + "." + resourceName
	result, ok := db[key]
	if result != nil {
		return *result, ok
	}
	return ResourceRow{}, false
}

func (db ResourcesMap) GetResourceID(ctx context.Context, section, resourceName string) (string, error) {
	key := section + "." + resourceName
	row, ok := db[key]
	if ok {
		return row.ResourceID, nil
	}
	return "", nil
}

func (db ResourcesMap) InsertResourcePre(ctx context.Context, section, resourceName, resourceID, config string) error {
	key := section + "." + resourceName
	db[key] = &ResourceRow{
		ResourceID: resourceID,
		Config:     config,
		PreTS:      time.Now().UnixNano(),
	}
	return nil
}

func (db ResourcesMap) UpdateResourcePre(ctx context.Context, section, resourceName, resourceID, config string) error {
	key := section + "." + resourceName
	row, ok := db[key]
	if !ok {
		row.ResourceID = resourceID
		db[key] = row
	}

	row.Config = config
	row.PreTS = time.Now().UnixNano()
	row.PostTS = 0
	return nil
}

func (db ResourcesMap) FinalizeResource(ctx context.Context, section, resourceName, resourceID string) error {
	key := section + "." + resourceName
	row, ok := db[key]
	if !ok {
		return errors.New("database error")
	}
	row.ResourceID = resourceID
	row.PostTS = time.Now().UnixNano()
	return nil
}

func LoadOrCreate(dir string) (*JSONFileDB, error) {
	result := &JSONFileDB{
		Path: filepath.Join(dir, "resourcedb.json"),
		DB: SimpleDB{
			Resources: make(ResourcesMap),
		},
	}
	err := result.Load()
	return result, err
}

func (db *JSONFileDB) Load() error {
	db.mu.Lock()
	defer db.mu.Unlock()

	data, err := os.ReadFile(db.Path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	if err := json.Unmarshal(data, &db.DB); err != nil {
		return err
	}

	return nil
}

func (db *JSONFileDB) unlockedSave() error {
	data, err := json.MarshalIndent(db.DB, "", " ")
	if err != nil {
		return err
	}

	return os.WriteFile(db.Path, data, 0o600)
}

func (db *JSONFileDB) GetRow(ctx context.Context, section, resourceName string) (ResourceRow, bool) {
	db.mu.Lock()
	defer db.mu.Unlock()

	return db.DB.Resources.GetRow(ctx, section, resourceName)
}

func (db *JSONFileDB) GetResourceID(ctx context.Context, section, resourceName string) (string, error) {
	db.mu.Lock()
	defer db.mu.Unlock()

	return db.DB.Resources.GetResourceID(ctx, section, resourceName)
}

func (db *JSONFileDB) InsertResourcePre(ctx context.Context, section, resourceName, resourceID, config string) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	err := db.DB.Resources.InsertResourcePre(ctx, section, resourceName, resourceID, config)
	if err != nil {
		return err
	}
	return db.unlockedSave()
}

func (db *JSONFileDB) UpdateResourcePre(ctx context.Context, section, resourceName, resourceID, config string) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	err := db.DB.Resources.UpdateResourcePre(ctx, section, resourceName, resourceID, config)
	if err != nil {
		return err
	}
	return db.unlockedSave()
}

func (db *JSONFileDB) FinalizeResource(ctx context.Context, section, resourceName, resourceID string) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	err := db.DB.Resources.FinalizeResource(ctx, section, resourceName, resourceID)
	if err != nil {
		return err
	}
	return db.unlockedSave()
}

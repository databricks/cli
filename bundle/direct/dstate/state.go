package dstate

import (
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/databricks/cli/bundle/deployplan"
	"github.com/databricks/cli/bundle/statemgmt/resourcestate"
	"github.com/databricks/cli/internal/build"
	"github.com/google/uuid"
)

const currentStateVersion = 2

type DeploymentState struct {
	Path    string
	Data    Database
	mu      sync.Mutex
	walFile *os.File
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

type WALHeader struct {
	Lineage      string `json:"lineage"`
	Serial       int    `json:"serial"`
	StateVersion int    `json:"state_version"`
	CLIVersion   string `json:"cli_version"`
}

type WALEntry struct {
	K string         `json:"k"`
	V *ResourceEntry `json:"v,omitempty"` // nil means delete
}

func NewDatabase(lineage string, serial int) Database {
	return Database{
		StateVersion: currentStateVersion,
		CLIVersion:   build.GetInfo().Version,
		Lineage:      lineage,
		Serial:       serial,
		State:        make(map[string]ResourceEntry),
	}
}

func (db *DeploymentState) SaveState(key, newID string, state any, dependsOn []deployplan.DependsOnEntry) error {
	db.AssertOpenedForWrite()
	db.mu.Lock()
	defer db.mu.Unlock()

	if db.Data.State == nil {
		db.Data.State = make(map[string]ResourceEntry)
	}

	jsonMessage, err := json.Marshal(state)
	if err != nil {
		return err
	}

	entry := ResourceEntry{
		ID:        newID,
		State:     json.RawMessage(jsonMessage),
		DependsOn: dependsOn,
	}

	db.Data.State[key] = entry
	return appendJSONLine(db.walFile, WALEntry{K: key, V: &entry})
}

func (db *DeploymentState) DeleteState(key string) error {
	db.AssertOpenedForWrite()
	db.mu.Lock()
	defer db.mu.Unlock()

	if db.Data.State == nil {
		return nil
	}

	delete(db.Data.State, key)
	return appendJSONLine(db.walFile, WALEntry{K: key})
}

func (db *DeploymentState) getResourceEntry(key string) (ResourceEntry, bool) {
	db.AssertOpened()
	db.mu.Lock()
	defer db.mu.Unlock()

	if db.Data.State == nil {
		return ResourceEntry{}, false
	}

	result, ok := db.Data.State[key]
	return result, ok
}

// GetResourceEntry returns the full resource entry for the given key.
func (db *DeploymentState) GetResourceEntry(key string) (ResourceEntry, bool) {
	return db.getResourceEntry(key)
}

// GetResourceID returns the ID of the resource for the given key, or an empty string if not found.
func (db *DeploymentState) GetResourceID(key string) string {
	entry, _ := db.getResourceEntry(key)
	return entry.ID
}

type (
	WithRecovery bool
	WithWrite    bool
)

func (db *DeploymentState) Open(ctx context.Context, path string, withRecovery WithRecovery, withWrite WithWrite) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	if db.Path != "" {
		panic(fmt.Sprintf("state already opened: %v, cannot open %v", db.Path, path))
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			// Not initializing lineage yet, we might have that saved in WAL
			db.Data = NewDatabase("", 0)
			db.Path = path
		} else {
			return err
		}
	} else {
		err = json.Unmarshal(data, &db.Data)
		if err != nil {
			return err
		}
		db.Path = path
	}

	walPath := walPath(db.Path)
	_, walError := os.Stat(walPath)
	if walError == nil {
		if withRecovery {
			err := db.mergeWalIntoState(ctx)
			if err != nil {
				return err
			}
		} else {
			return fmt.Errorf("unprocessed WAL exists: %s", walPath)
		}
	}

	if err := migrateState(&db.Data); err != nil {
		return fmt.Errorf("migrating state %s: %w", path, err)
	}

	if withWrite {
		db.walFile, err = os.OpenFile(walPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
		if err != nil {
			return fmt.Errorf("failed to open WAL file %s: %w", walPath, err)
		}
		lineage := db.Data.Lineage
		if lineage == "" {
			lineage = uuid.New().String()
		}
		// Set our Serial to the next one
		db.Data.Serial += 1
		walHead := WALHeader{
			Lineage:      lineage,
			Serial:       db.Data.Serial, // next serial
			StateVersion: currentStateVersion,
			CLIVersion:   build.GetInfo().Version,
		}
		err := appendJSONLine(db.walFile, walHead)
		if err != nil {
			return err
		}
	}

	return nil
}

func (db *DeploymentState) mergeWalIntoState(ctx context.Context) error {
}

func (db *DeploymentState) Finalize(ctx context.Context) error {
	db.AssertOpenedForWrite()
	db.mu.Lock()
	defer db.mu.Unlock()

	db.walFile.Close()
	db.walFile = nil
	return db.mergeWalIntoState(ctx)
}

// Close closes the WAL file without saving state.
func (db *DeploymentState) Close() error {
	db.mu.Lock()
	defer db.mu.Unlock()

	if db.wal != nil {
		if err := db.wal.close(); err != nil {
			return err
		}
		db.wal = nil
	}
	return nil
}

func (db *DeploymentState) AssertOpened() {
	if db.Path == "" {
		panic("internal error: DeploymentState must be opened first")
	}
}

func (db *DeploymentState) AssertOpenedForWrite() {
	db.AssertOpened()
	if db.walFile == nil {
		panic("internal error: DeploymentState must be opened in write mode")
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
	db.AssertOpenedForWrite()
	data, err := json.MarshalIndent(db.Data, "", " ")
	if err != nil {
		return err
	}

	dir := filepath.Dir(db.Path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("failed to create directory %#v: %w", dir, err)
	}

	err = os.WriteFile(db.Path, data, 0o600)
	if err != nil {
		return fmt.Errorf("failed to save resources state to %#v: %w", db.Path, err)
	}

	return nil
}

func appendJSONLine(file *os.File, obj any) error {
	data, err := json.Marshal(obj)
	if err != nil {
		return err
	}
	data = append(data, '\n')

	_, err = file.Write(data)
	// no fsync here, not needed
	return err
}

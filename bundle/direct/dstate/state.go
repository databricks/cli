package dstate

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
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
	Path             string
	Data             Database
	mu               sync.Mutex
	wal              *WAL
	recoveredFromWAL bool
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
	db.AssertOpened()
	db.mu.Lock()
	defer db.mu.Unlock()

	if db.Data.State == nil {
		db.Data.State = make(map[string]ResourceEntry)
	}

	jsonMessage, err := json.MarshalIndent(state, "", " ")
	if err != nil {
		return err
	}

	entry := ResourceEntry{
		ID:        newID,
		State:     json.RawMessage(jsonMessage),
		DependsOn: dependsOn,
	}

	if err := db.ensureWALOpen(); err != nil {
		return fmt.Errorf("failed to open WAL: %w", err)
	}
	if err := db.wal.writeJSON(WALEntry{K: key, V: &entry}); err != nil {
		return fmt.Errorf("failed to write WAL entry: %w", err)
	}

	db.Data.State[key] = entry

	return nil
}

func (db *DeploymentState) DeleteState(key string) error {
	db.AssertOpened()
	db.mu.Lock()
	defer db.mu.Unlock()

	if db.Data.State == nil {
		return nil
	}

	if err := db.ensureWALOpen(); err != nil {
		return fmt.Errorf("failed to open WAL: %w", err)
	}
	if err := db.wal.writeJSON(WALEntry{K: key}); err != nil {
		return fmt.Errorf("failed to write WAL entry: %w", err)
	}

	delete(db.Data.State, key)

	return nil
}

func (db *DeploymentState) ensureWALOpen() error {
	if db.wal != nil {
		return nil
	}

	wal, err := openWAL(db.Path)
	if err != nil {
		return err
	}

	lineage := db.Data.Lineage
	if lineage == "" {
		lineage = uuid.New().String()
		db.Data.Lineage = lineage
	}

	walSerial := db.Data.Serial + 1

	if err := wal.writeJSON(WALHeader{Lineage: lineage, Serial: walSerial}); err != nil {
		wal.close()
		return err
	}

	db.wal = wal
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

func (db *DeploymentState) Open(ctx context.Context, path string) error {
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
			db.Data = NewDatabase("", 0)
			db.Path = path
			if err := db.unlockedSave(); err != nil {
				return err
			}
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

	recovered, err := recoverFromWAL(ctx, path, &db.Data)
	if err != nil {
		return fmt.Errorf("WAL recovery failed: %w", err)
	}

	if err := migrateState(&db.Data); err != nil {
		return fmt.Errorf("migrating state %s: %w", path, err)
	}

	if recovered {
		if err := db.unlockedSave(); err != nil {
			return err
		}
		if err := cleanupWAL(path); err != nil {
			return err
		}
		db.recoveredFromWAL = true
	}
	return nil
}

func (db *DeploymentState) Finalize() error {
	db.mu.Lock()
	defer db.mu.Unlock()

	hadOpenWAL := db.wal != nil
	if hadOpenWAL {
		if err := db.wal.close(); err != nil {
			return err
		}
		db.wal = nil

		validationDB := db.Data
		replayResult, err := replayWAL(db.Path, &validationDB)
		if err != nil {
			return fmt.Errorf("failed to replay WAL during finalize: %w", err)
		}
		if !replayResult.recovered {
			return errors.New("failed to replay WAL during finalize: WAL file not found or stale")
		}
		if len(replayResult.corruptedEntries) > 0 {
			first := replayResult.corruptedEntries[0]
			return fmt.Errorf("failed to replay WAL during finalize: corrupted entry at line %d: %v", first.lineNumber, first.parseErr)
		}
	}

	if db.Data.Lineage == "" && !hadOpenWAL && len(db.Data.State) == 0 {
		return nil
	}

	if db.Data.Lineage == "" {
		db.Data.Lineage = uuid.New().String()
	}

	db.Data.Serial++

	if err := db.unlockedSave(); err != nil {
		return err
	}

	if hadOpenWAL {
		if err := cleanupWAL(db.Path); err != nil {
			return err
		}
	}

	return nil
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

// RecoveredFromWAL reports whether Open recovered state from the WAL.
func (db *DeploymentState) RecoveredFromWAL() bool {
	return db.recoveredFromWAL
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
